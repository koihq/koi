# Onboarding

Both setup paths have four steps after sign-up, and the fourth is the first real answer.

| Step | Person | Agent |
| --- | --- | --- |
| 1 | Choose | Choose |
| 2 | Website | Connect |
| 3 | Confirm the name and country | Paste the prompt |
| 4 | First answers | First answers |

The name on step 3 comes from `GET /api/v1/brands/lookup`, which reads the public website or GitHub
repository. Website requests connect only to public addresses, checked after DNS resolution, so a
hostname or redirect cannot reach the server's own network. A failed lookup falls back to a name
parsed from the URL, without claiming it came from the site.

While the draft runs, Koi shows back what its research understood (`GenerationStatus.summary`). The
plan under the result is counted from answers that left the brand out; it contains no generated
advice. See [billing](billing.md) for the First Look limits.

## Funnel

Koi Cloud records the first time an organization reaches each step in `onboarding_events`. It stores
the step name and nothing about the brand or its answers.

```sql
SELECT step, count(*) AS organizations
FROM onboarding_events
GROUP BY step
ORDER BY organizations DESC;
```

Time from brand creation to the first real answer:

```sql
SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY seconds) AS median_seconds
FROM (
  SELECT extract(epoch FROM min(results.fetched_at) - brands.created_at) AS seconds
  FROM brands
  JOIN visibility_runs AS runs ON runs.brand_id = brands.id AND runs.fast_start
  JOIN visibility_results AS results ON results.run_id = runs.id
    AND results.outcome IN ('cited', 'mentioned', 'miss')
  GROUP BY brands.id
) AS first_answers;
```
