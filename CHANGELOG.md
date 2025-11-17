# Changelog

## 1.0.0 (2025-11-17)


### Features

* **concurrency:** add context, fix builders and add more tests ([fc00919](https://github.com/u-ctf/controller-fwk/commit/fc0091978a14f28d572afaacc104618ab2d6ebbf))
* **context:** update context to be more generic ([352cff0](https://github.com/u-ctf/controller-fwk/commit/352cff053eb29bef43ab7603284bdbbc4ce37ec8))
* **controller:** add pausing feature ([208369e](https://github.com/u-ctf/controller-fwk/commit/208369e347e3013a6742dbc464edb7b979b8fae2))
* **finalizer:** add first try at finalizer ([d5f085b](https://github.com/u-ctf/controller-fwk/commit/d5f085bb14e8b3f7dd6173a6c8618b87f39ba9d1))
* first version ([fc6a527](https://github.com/u-ctf/controller-fwk/commit/fc6a527ea637c37c01b714d826774476f57afa7c))
* **instrumentation:** rename tracing to instrumentation and improve the design ([202d365](https://github.com/u-ctf/controller-fwk/commit/202d36594748a5ef3105ec369e8358e9c1d21944))
* **tests:** add more e2e tests ([6477a60](https://github.com/u-ctf/controller-fwk/commit/6477a60dd9822a7938e2dd0b3630df9ca6a4d2d5))
* **v2:** add builder pattern and migrate to it in tests ([dd6b0e7](https://github.com/u-ctf/controller-fwk/commit/dd6b0e7a10e8eebe6ac889ddb9d1025879989861))
* **v2:** modify naming and some inner workings for simplicity ([400c2dd](https://github.com/u-ctf/controller-fwk/commit/400c2dd97a196db00aeaa07caf370d6772512bcd))
* **v2:** working towards v2 ([8c42546](https://github.com/u-ctf/controller-fwk/commit/8c4254663bb175de80af6a4eebdda6599faafaef))


### Bug Fixes

* **context:** export Data field ([e644b68](https://github.com/u-ctf/controller-fwk/commit/e644b689dbd5d054126e00a62a9055d552eb9c42))
* **e2e:** create ns using kubectl instead of clientset used for the operator ([534c3a6](https://github.com/u-ctf/controller-fwk/commit/534c3a655b5494a890673303c6131afc47b96fcb))
* **e2e:** remove systemd from e2e image ([3acf50e](https://github.com/u-ctf/controller-fwk/commit/3acf50efe3db9984aabb677ef35ae0448cc1aef7))
* **e2e:** use patch instead of update ([bfc4c38](https://github.com/u-ctf/controller-fwk/commit/bfc4c38d4b58d2cc9083cfaa669d47ac5f62ebb4))
* **mgr:** don't require mgr to be exposed by default, it's only needed when autowatch is set ([5787340](https://github.com/u-ctf/controller-fwk/commit/5787340bb4613efaecdf8f97b65a2c087e3c7638))
* **release:** change settings and remove 0.1.0 ([a38692c](https://github.com/u-ctf/controller-fwk/commit/a38692cc9a0530bfca5c94e7b18e677d0e14273d))
* **sample-operator:** change typing to use alias and reduce boilerplate ([7a5ba48](https://github.com/u-ctf/controller-fwk/commit/7a5ba483dbd89e7af150c3cc2ce08f5a5267a16f))
* **tests:** cleanup labels to avoid ns timeout ([3bc1847](https://github.com/u-ctf/controller-fwk/commit/3bc18476a425b37c15abb8c29c98c7ff2a288fb2))
* **tests:** implement fixes by copilot ([eb637b5](https://github.com/u-ctf/controller-fwk/commit/eb637b59f62f8739ff5c6c30855893f8cb99f012))

## [1.0.0](https://github.com/u-ctf/controller-fwk/releases/tag/v1.0.0) - 2025-11-13

### ❤️ Thanks to all contributors! ❤️

@yyewolf

### 📈 Enhancement

- chore: update godoc in most places [[#10](https://github.com/u-ctf/controller-fwk/pull/10)]
- fix(context): export Data field [[#9](https://github.com/u-ctf/controller-fwk/pull/9)]
- fix(sample-operator): change typing to use alias and reduce boilerplate [[#8](https://github.com/u-ctf/controller-fwk/pull/8)]
- feat(context): update context to be more generic [[#7](https://github.com/u-ctf/controller-fwk/pull/7)]
- feat(concurrency): add context, fix builders and add more tests [[#6](https://github.com/u-ctf/controller-fwk/pull/6)]

### Misc

- break: go to 1.0.0 ([35f4515](https://github.com/u-ctf/controller-fwk/commit/35f45154284e93b881d77c924a28eb4fcb782530))
- chore: update godoc in most places ([91d112c](https://github.com/u-ctf/controller-fwk/commit/91d112c84b899dbb4d115860359a8704fa8bd552))
- fix(context): export Data field ([05ddbfb](https://github.com/u-ctf/controller-fwk/commit/05ddbfb77a278e6a7b7349ee7b7c447505baabac))
- fix(sample-operator): change typing to use alias and reduce boilerplate ([b063b6f](https://github.com/u-ctf/controller-fwk/commit/b063b6f24b1cd8bab5a7a9d5b49115992369d651))
- feat(context): update context to be more generic ([c219fef](https://github.com/u-ctf/controller-fwk/commit/c219fefc362daba70f25fd52495edac2985dcec1))
- ci: use wrapping technique for super secret auth ([c93a5b9](https://github.com/u-ctf/controller-fwk/commit/c93a5b914bdad0c93cfebb75542ce649d8455d34))
- ci: add tests coverage ([55b8330](https://github.com/u-ctf/controller-fwk/commit/55b833020b4700d4dca4e7d6424ab627de883960))
- ci: make tests faster via parallelization ([d5f9407](https://github.com/u-ctf/controller-fwk/commit/d5f9407632836ee36faa0ea9b018b981adfa3962))
- feat(concurrency): add context, fix builders and add more tests ([d3310a7](https://github.com/u-ctf/controller-fwk/commit/d3310a7fdc319fb70c30c07757620be6d0d04960))
- ci: update to be safer ([2d1356f](https://github.com/u-ctf/controller-fwk/commit/2d1356fe231fd74752854fd132568aa4b5ca9d75))
- ci: add ready-release-go ([b8f2390](https://github.com/u-ctf/controller-fwk/commit/b8f2390bb9048dd3bf39cd69b1992ab6cbaa11b5))
- refactor(instrumentation): transform into a otel pluggable implementation ([1f80b37](https://github.com/u-ctf/controller-fwk/commit/1f80b377dfffb86d2e551ad3e5c0e018099c88eb))
- fix(mgr): don't require mgr to be exposed by default, it's only needed when autowatch is set ([5787340](https://github.com/u-ctf/controller-fwk/commit/5787340bb4613efaecdf8f97b65a2c087e3c7638))
- fix(e2e): use patch instead of update ([bfc4c38](https://github.com/u-ctf/controller-fwk/commit/bfc4c38d4b58d2cc9083cfaa669d47ac5f62ebb4))
- chore(readme): update readme for practices ([7adad73](https://github.com/u-ctf/controller-fwk/commit/7adad73c102cdab54db1c64b706c0eb020b8019e))
- fix(e2e): remove systemd from e2e image ([3acf50e](https://github.com/u-ctf/controller-fwk/commit/3acf50efe3db9984aabb677ef35ae0448cc1aef7))
- fix(e2e): create ns using kubectl instead of clientset used for the operator ([534c3a6](https://github.com/u-ctf/controller-fwk/commit/534c3a655b5494a890673303c6131afc47b96fcb))
- feat(v2): add builder pattern and migrate to it in tests ([dd6b0e7](https://github.com/u-ctf/controller-fwk/commit/dd6b0e7a10e8eebe6ac889ddb9d1025879989861))
- feat(finalizer): add first try at finalizer ([d5f085b](https://github.com/u-ctf/controller-fwk/commit/d5f085bb14e8b3f7dd6173a6c8618b87f39ba9d1))
- feat(tests): add more e2e tests ([6477a60](https://github.com/u-ctf/controller-fwk/commit/6477a60dd9822a7938e2dd0b3630df9ca6a4d2d5))
- feat(v2): modify naming and some inner workings for simplicity ([400c2dd](https://github.com/u-ctf/controller-fwk/commit/400c2dd97a196db00aeaa07caf370d6772512bcd))
- feat(v2): working towards v2 ([8c42546](https://github.com/u-ctf/controller-fwk/commit/8c4254663bb175de80af6a4eebdda6599faafaef))
- tests: add a bit more tests ([6d35318](https://github.com/u-ctf/controller-fwk/commit/6d3531890bbaf5f6df1e2d654420aac4826209e8))
- feat(instrumentation): rename tracing to instrumentation and improve the design ([202d365](https://github.com/u-ctf/controller-fwk/commit/202d36594748a5ef3105ec369e8358e9c1d21944))
- feat: first version ([fc6a527](https://github.com/u-ctf/controller-fwk/commit/fc6a527ea637c37c01b714d826774476f57afa7c))
