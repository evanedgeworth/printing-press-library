# Acceptance Report: janeapp

Level: Quick Check (live public endpoints; auth-gated verified for graceful handling)
Gate: PASS (8/8)

Verified live against embophysio.janeapp.com and leahkangas.janeapp.com:
1. clinic add / list / use — multi-tenant store works; --clinic routes to the right subdomain.
2. locations (both clinics) — real data (Embo Physio address; Leah Kangas LMT).
3. treatments — real services with book_online flag, price, duration.
4. staff — practitioner with all_treatment_ids, allow_online_booking.
5. next-opening (flagship) — found real 2026-07-08T08:30 slot for treatment 1 / staff 1, paging past the 7-day window cap.
6. appointments upcoming (no session) — clean 401 handling with actionable hint.
7. book (no session) — clear "not logged in" error; dry-run-by-default write path.
8. doctor — config/API reachability OK; clinic-aware base URL.

Pending live (needs a patient session — see README Known Gaps):
- Two-step username/password auth handshake end-to-end.
- appointments/agenda authed shape; book/reschedule/cancel real submit (dry-run default + --confirm gate).

Printing Press issues for retro: none material; scorecard auth_protocol heuristic (2/10) does not model cookie-session-via-2-step-form auth, which is Jane's real mechanism.
