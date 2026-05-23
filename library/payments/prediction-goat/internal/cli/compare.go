// Copyright 2026 mvanhorn. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/payments/prediction-goat/internal/store"
)

type comparePair struct {
	Topic    string        `json:"topic"`
	Match    float64       `json:"match,omitempty"`
	PM       *compareVenue `json:"polymarket,omitempty"`
	Kalshi   *compareVenue `json:"kalshi,omitempty"`
	DeltaPct *float64      `json:"deltaPct,omitempty"`
}

type compareVenue struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	YesProbability float64 `json:"yesProbability"`
	YesPercent     float64 `json:"yesPercent,omitempty"`
	Volume24h      float64 `json:"volume24h"`
	EndDate        string  `json:"endDate"`
	URL            string  `json:"url"`
	Untraded       bool    `json:"untraded,omitempty"`
}

type compareResult struct {
	Topic string        `json:"topic"`
	Pairs []comparePair `json:"pairs"`
}

type rawMarket struct {
	Venue          string
	ID             string
	Title          string
	YesProbability float64
	YesAsk         float64
	NoAsk          float64
	LastPrice      float64
	Volume24h      float64
	EndDate        string
	URL            string
	Untraded       bool
}

func newCompareCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var dbPath string
	cmd := &cobra.Command{
		Use:   "compare <topic>",
		Short: "Side-by-side Polymarket and Kalshi prices for a topic",
		Example: `  prediction-goat-pp-cli compare election --json
  prediction-goat-pp-cli compare 'arizona basketball' --limit 5`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if limit < 1 {
				return fmt.Errorf("compare: --limit must be greater than zero")
			}
			if dbPath == "" {
				dbPath = defaultDBPath("prediction-goat-pp-cli")
			}
			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("compare: %w", err)
			}
			defer db.Close()

			topic := strings.Join(args, " ")
			searchLimit := limit * 10
			if searchLimit < 50 {
				searchLimit = 50
			}
			pmMarkets, kalshiMarkets, err := loadCompareMarkets(cmd, db, topic, searchLimit)
			if err != nil {
				return fmt.Errorf("compare: %w", err)
			}
			pairs := pairCompareMarkets(topic, pmMarkets, kalshiMarkets, limit)
			result := compareResult{Topic: topic, Pairs: pairs}
			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				if err := printJSONFiltered(cmd.OutOrStdout(), result, flags); err != nil {
					return err
				}
			} else if err := printCompareTable(cmd.OutOrStdout(), pairs); err != nil {
				return err
			}
			if len(pairs) == 0 {
				return notFoundErr(fmt.Errorf("no Polymarket-Kalshi market pairs found for topic %q (try a broader query, or sync more data first)", topic))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Max pairs returned")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: standard cache location)")
	return cmd
}

func loadCompareMarkets(cmd *cobra.Command, db *store.Store, topic string, limit int) ([]rawMarket, []rawMarket, error) {
	rows, err := db.DB().QueryContext(cmd.Context(), `SELECT r.resource_type, r.id, r.data FROM resources r
JOIN resources_fts f ON r.id = f.id AND r.resource_type = f.resource_type
WHERE resources_fts MATCH ?
AND r.resource_type IN ('markets','kalshi_markets')
ORDER BY rank LIMIT ?`, topicFTSQuery(topic), limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	pmMarkets := make([]rawMarket, 0)
	kalshiMarkets := make([]rawMarket, 0)
	for rows.Next() {
		var typ, id, data sql.NullString
		if err := rows.Scan(&typ, &id, &data); err != nil {
			return nil, nil, err
		}
		if !typ.Valid || !data.Valid {
			continue
		}
		market, ok := rawMarketFromJSON(typ.String, id.String, data.String)
		if !ok {
			continue
		}
		if market.Venue == "polymarket" {
			pmMarkets = append(pmMarkets, market)
		} else {
			kalshiMarkets = append(kalshiMarkets, market)
		}
	}
	return pmMarkets, kalshiMarkets, rows.Err()
}

func pairCompareMarkets(topic string, pmMarkets, kalshiMarkets []rawMarket, limit int) []comparePair {
	pairs := make([]comparePair, 0)
	usedKalshi := make(map[int]bool)
	for _, pm := range pmMarkets {
		bestIdx := -1
		bestScore := 0.0
		for i, kalshi := range kalshiMarkets {
			if usedKalshi[i] {
				continue
			}
			if score := tokenJaccard(pm.Title, kalshi.Title); score > bestScore {
				bestIdx = i
				bestScore = score
			}
		}
		pmVenue := compareVenueFromRaw(pm)
		pair := comparePair{Topic: topic, PM: &pmVenue}
		if bestIdx >= 0 && bestScore >= 0.20 {
			usedKalshi[bestIdx] = true
			kalshiVenue := compareVenueFromRaw(kalshiMarkets[bestIdx])
			delta := (pm.YesProbability - kalshiMarkets[bestIdx].YesProbability) * 100
			pair.Kalshi = &kalshiVenue
			pair.Match = bestScore
			pair.DeltaPct = &delta
		}
		pairs = append(pairs, pair)
		if len(pairs) >= limit {
			return pairs
		}
	}
	for i, kalshi := range kalshiMarkets {
		if usedKalshi[i] {
			continue
		}
		kalshiVenue := compareVenueFromRaw(kalshi)
		pairs = append(pairs, comparePair{Topic: topic, Kalshi: &kalshiVenue})
		if len(pairs) >= limit {
			return pairs
		}
	}
	return pairs
}

func rawMarketFromJSON(resourceType, fallbackID, raw string) (rawMarket, bool) {
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return rawMarket{}, false
	}
	switch resourceType {
	case "markets":
		id := firstNonEmpty(jsonString(obj, "slug"), fallbackID)
		yes := jsonFloat(obj, "lastTradePrice")
		return rawMarket{Venue: "polymarket", ID: id, Title: firstNonEmpty(jsonString(obj, "question"), jsonString(obj, "title")), YesProbability: yes, LastPrice: yes, Volume24h: firstFloat(obj, "volume24h", "volume24hr", "volumeNum"), EndDate: jsonString(obj, "endDate"), URL: "https://polymarket.com/market/" + id}, true
	case "kalshi_markets":
		id := firstNonEmpty(jsonString(obj, "ticker"), fallbackID)
		eventTicker := jsonString(obj, "event_ticker")
		title := jsonString(obj, "title")
		// Multi-Variable Event markets carry comma-concatenated YES/NO
		// outcome legs in the `title` field; fall back to the parent
		// event ticker when that pattern is detected so downstream
		// comparison/matching uses a stable string.
		if looksLikeMultiLegTitle(title) && eventTicker != "" {
			title = eventTicker
		}
		yesAsk := jsonFloat(obj, "yes_ask_dollars")
		noAsk := jsonFloat(obj, "no_ask_dollars")
		lastPrice := jsonFloat(obj, "last_price_dollars")
		volume24h := jsonFloat(obj, "volume_24h_fp")
		untraded := isUntradedKalshi(yesAsk, noAsk, lastPrice, volume24h)
		return rawMarket{Venue: "kalshi", ID: id, Title: title, YesProbability: lastPrice, YesAsk: yesAsk, NoAsk: noAsk, LastPrice: lastPrice, Volume24h: volume24h, EndDate: jsonString(obj, "expiration_time"), URL: "https://kalshi.com/markets/" + eventTicker + "/" + id, Untraded: untraded}, true
	default:
		return rawMarket{}, false
	}
}

func compareVenueFromRaw(m rawMarket) compareVenue {
	return compareVenue{ID: m.ID, Title: m.Title, YesProbability: m.YesProbability, YesPercent: yesPercent(m.YesProbability), Volume24h: m.Volume24h, EndDate: m.EndDate, URL: m.URL, Untraded: m.Untraded}
}

func printCompareTable(w io.Writer, pairs []comparePair) error {
	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "Match\tPM Title\tKalshi Title\tPM%\tKalshi%\tDelta")
	for _, pair := range pairs {
		pmTitle, pmProb := "", ""
		kalshiTitle, kalshiProb := "", ""
		if pair.PM != nil {
			pmTitle = truncate(pair.PM.Title, 48)
			pmProb = formatProb(pair.PM.YesProbability)
		}
		if pair.Kalshi != nil {
			kalshiTitle = truncate(pair.Kalshi.Title, 48)
			kalshiProb = formatProb(pair.Kalshi.YesProbability)
		}
		delta := ""
		if pair.DeltaPct != nil {
			delta = fmt.Sprintf("%+.1f", *pair.DeltaPct)
		}
		fmt.Fprintf(tw, "%.2f\t%s\t%s\t%s\t%s\t%s\n", pair.Match, pmTitle, kalshiTitle, pmProb, kalshiProb, delta)
	}
	return tw.Flush()
}
