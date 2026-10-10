## 1. Registry guard test

- [x] 1.1 Add `web/src/lib/collections.test.ts` (vitest) asserting every `FILTER_COLLECTIONS` entry has a unique `slug` and a non-empty `params` map (the invariant guarding all subsequent data tasks)

## 2. Tech-category landings

- [x] 2.1 Add curated tech-category filter collections (`category=` param): backend, frontend, fullstack, devops, sre, data-engineering, data-science, machine-learning, mobile, security, qa, architecture, embedded, network-engineering — each with a hand-written title/description and a live open-job count ≥ 300 verified before adding; re-run the guard test

## 3. Seniority landings

- [x] 3.1 Add curated seniority filter collections (`seniority=` param): junior, senior, lead, staff, principal, intern — hand-written copy, count verified; re-run the guard test

## 4. Infra-skill landings

- [x] 4.1 Add curated infra-skill filter collections (`skills=` param): aws, kubernetes, terraform, docker, postgresql, redis, kafka, graphql — hand-written copy, count verified; re-run the guard test

## 5. Named-role landings

- [x] 5.1 Add curated named tech-role filter collections (`role=` param): software-engineer plus any skill×seniority combos whose live count is individually verified ≥ 300 (drop thin ones); re-run the guard test

## 6. Verify and ship

- [x] 6.1 `svelte-check` and `vitest` green; spot-check 2–3 new landing pages render a scoped feed with a non-zero count and a self-canonical
