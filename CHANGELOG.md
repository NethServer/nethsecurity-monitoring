# Changelog

## [1.1.2](https://github.com/NethServer/nethsecurity-monitoring/compare/v1.1.1...v1.1.2) (2026-05-06)


### Bug Fixes

* removed directory creation ([08e0aa2](https://github.com/NethServer/nethsecurity-monitoring/commit/08e0aa237bcaec61f80ea05e27ea0da22e9da323))

## [1.1.1](https://github.com/NethServer/nethsecurity-monitoring/compare/v1.1.0...v1.1.1) (2026-05-05)


### Bug Fixes

* concurrent resolution ([679e6ad](https://github.com/NethServer/nethsecurity-monitoring/commit/679e6adf743ff754d194c8732a735e515bf54b0e))
* creating directories if missing ([601fdbd](https://github.com/NethServer/nethsecurity-monitoring/commit/601fdbdb4dd0fba53067a4f0143dedbe11ec3287))
* **deps:** update module modernc.org/sqlite to v1.50.0 ([#20](https://github.com/NethServer/nethsecurity-monitoring/issues/20)) ([62db32a](https://github.com/NethServer/nethsecurity-monitoring/commit/62db32ade8b4d9131f85bee97a74059afc0f2d2a))


### Reverts

* "fix(deps): update module modernc.org/sqlite to v1.50.0 ([#20](https://github.com/NethServer/nethsecurity-monitoring/issues/20))" ([eee7067](https://github.com/NethServer/nethsecurity-monitoring/commit/eee70672409a509eb0331113d691f1da8c6c7b4b))

## [1.1.0](https://github.com/NethServer/nethsecurity-monitoring/compare/v1.0.2...v1.1.0) (2026-04-30)


### Features

* added ns-stats binary ([994f92f](https://github.com/NethServer/nethsecurity-monitoring/commit/994f92f595e76c3c673ffe2c52035888b5b19cfa))


### Bug Fixes

* avoiding restart on socket reload ([3631e60](https://github.com/NethServer/nethsecurity-monitoring/commit/3631e60850ed628f46d269825b04504c661daf96))
* **deps:** update module github.com/gofiber/fiber/v2 to v2.52.13 ([#18](https://github.com/NethServer/nethsecurity-monitoring/issues/18)) ([22a0ab0](https://github.com/NethServer/nethsecurity-monitoring/commit/22a0ab058853df5678669af52519ce4672b5eccc))

## [1.0.2](https://github.com/NethServer/nethsecurity-monitoring/compare/v1.0.1...v1.0.2) (2026-03-24)


### Bug Fixes

* decoding all fields from documentation ([affb289](https://github.com/NethServer/nethsecurity-monitoring/commit/affb2895a86ad2f4035dcaedde979ebbee6bea97))
* using only completed events ([a7712be](https://github.com/NethServer/nethsecurity-monitoring/commit/a7712be6b289d839320d2c68beeb25778eb7886e))

## [1.0.1](https://github.com/NethServer/nethsecurity-monitoring/compare/v1.0.0...v1.0.1) (2026-03-12)


### Bug Fixes

* avoid exiting on malformed flow ([e147de9](https://github.com/NethServer/nethsecurity-monitoring/commit/e147de9bc92ab17447648d6bc5658a3a6592fd4e))
* **deps:** update module github.com/gofiber/fiber/v2 to v2.52.12 [security] ([#11](https://github.com/NethServer/nethsecurity-monitoring/issues/11)) ([78aa2f9](https://github.com/NethServer/nethsecurity-monitoring/commit/78aa2f92212dbb6fd2ac270fb42017de3075d589))

## 1.0.0 (2026-03-06)


### Features

* added API endpoint for flows fetching ([#10](https://github.com/NethServer/nethsecurity-monitoring/issues/10)) ([570c628](https://github.com/NethServer/nethsecurity-monitoring/commit/570c628dea88179505db13af8e55d598aa967981))
* added configurable discard of expired flows ([21cb931](https://github.com/NethServer/nethsecurity-monitoring/commit/21cb931b8edf54d78edfaa51d53f13640e47f668))
* added sync for subprocess clean closing ([21cb931](https://github.com/NethServer/nethsecurity-monitoring/commit/21cb931b8edf54d78edfaa51d53f13640e47f668))
* initial release ([#3](https://github.com/NethServer/nethsecurity-monitoring/issues/3)) ([d0d05c2](https://github.com/NethServer/nethsecurity-monitoring/commit/d0d05c2306d78dacecfaf5e272e489f4956faab6))


### Performance Improvements

* improving performance using purging ([a997874](https://github.com/NethServer/nethsecurity-monitoring/commit/a9978744f63568e2ea10e225bae901cb7271c7e5))
