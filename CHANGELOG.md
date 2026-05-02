# Changelog

## 1.0.0 (2026-05-02)


### Features

* **add:** improve path resolution for files within dotfiles directory ([90f0e8c](https://github.com/satocchi0416sh/dotgo/commit/90f0e8c7941cfcec710aa6ea477a3ec81e867d3d))
* **cmd:** add interactive UI with progress indicators and confirmations ([d7eca65](https://github.com/satocchi0416sh/dotgo/commit/d7eca6500a91afe8e86d0f9b1b9e100f76a3e6b9))
* **cmd:** auto-fallback to verbose mode when stdin is not a tty ([2f19120](https://github.com/satocchi0416sh/dotgo/commit/2f19120a44cc365f813ff0745065c5e4e736e18f))
* **cmd:** auto-fallback to verbose mode when stdin is not a tty ([e9042ea](https://github.com/satocchi0416sh/dotgo/commit/e9042eade3cbcad2f19e70871cca6bcbd675e4ca))
* **cmd:** render dotgo status as path-hierarchy tree ([0411b3b](https://github.com/satocchi0416sh/dotgo/commit/0411b3bc15f797ecde72afb17fd275639f7f380c))
* **cmd:** render dotgo status as path-hierarchy tree with --flat fallback ([488b66e](https://github.com/satocchi0416sh/dotgo/commit/488b66e3151c49800e1677de5bc41a24fe53d406))
* **config:** add manifest-based configuration with tag filtering ([8fc23cb](https://github.com/satocchi0416sh/dotgo/commit/8fc23cb9cab1107c9f5f0098387b2f76fdf98e6d))
* **engine:** add core engine for dotfile operations ([82f49c0](https://github.com/satocchi0416sh/dotgo/commit/82f49c0658f4e6c80bd26c9eb9376cee01963531))
* **engine:** support directory sources in dotgo add via PathExists + recursive copy ([375e8e3](https://github.com/satocchi0416sh/dotgo/commit/375e8e3607bced76be0f236350d8cc50dd01cf64))
* **template:** add secure template processing with secret management ([087eb3c](https://github.com/satocchi0416sh/dotgo/commit/087eb3c75b6010ef9e69ed1363e67aa8b8dcc04e))
* **utils:** add file and path utility functions ([2f58725](https://github.com/satocchi0416sh/dotgo/commit/2f587256d30147d20f5148d9ea5d9f2df602ee87))
* **utils:** add PathExists for file/dir/symlink existence checks ([81783a1](https://github.com/satocchi0416sh/dotgo/commit/81783a12d152b9a8b13c44422da147b910b00a0d))


### Bug Fixes

* **apply:** eliminate TUI/log interleave and N×N reapply duplication ([2b3fe5f](https://github.com/satocchi0416sh/dotgo/commit/2b3fe5f3a732e396f8ca1e6ed1697eab35086aa3))
* **apply:** eliminate TUI/log interleave and N×N reapply duplication ([1deefd9](https://github.com/satocchi0416sh/dotgo/commit/1deefd911970033b897534e2f63fbe42fdd2104d))
* **build:** use full github module path for go install compatibility ([e5b0f06](https://github.com/satocchi0416sh/dotgo/commit/e5b0f0679ddb7a6e02f050d34c676ce7f8d74890))
* **cmd:** read version from runtime/debug.ReadBuildInfo instead of hardcoded literal ([#6](https://github.com/satocchi0416sh/dotgo/issues/6)) ([fb1e3e1](https://github.com/satocchi0416sh/dotgo/commit/fb1e3e12458f80b255c0c6ef13d0873a0d780105))
* **config,engine:** triage pre-existing test failures ([f779529](https://github.com/satocchi0416sh/dotgo/commit/f779529f8d7ebdfb3e5747bc370dd50386be0622))
* **config,engine:** triage pre-existing test failures (5 fixes + 2 cascade) ([7c98bf9](https://github.com/satocchi0416sh/dotgo/commit/7c98bf9ba5f3397a859e7f9f583656d439721b1a))
* **engine:** allow directory sources in dotgo apply ([5faaa86](https://github.com/satocchi0416sh/dotgo/commit/5faaa86d3c22171b1c2d84150c06a85ba00073b8))
* **engine:** detect symlink-to-directory targets in dotgo rm ([868c32e](https://github.com/satocchi0416sh/dotgo/commit/868c32e4444e122b1c3a57726e3bc504de2e4bd9))
* **engine:** support directory sources in backup/restore via os.Rename ([e0a271c](https://github.com/satocchi0416sh/dotgo/commit/e0a271cf2ec5e27ea93e5ee119f3901360cf6e3f))
* **lint:** handle viper.BindPFlag errors and add minimal golangci config ([64fb6e3](https://github.com/satocchi0416sh/dotgo/commit/64fb6e39f10aa305bc9a88d73e54249f0bfb2ce4))
* **release:** drop unsupported package-name input from release-please-action ([#8](https://github.com/satocchi0416sh/dotgo/issues/8)) ([0fa065c](https://github.com/satocchi0416sh/dotgo/commit/0fa065c43bfba668c308d03c32a8136cc7267269))
* **ui:** align spinner color with the 4-color palette (245 → 241) ([45ec366](https://github.com/satocchi0416sh/dotgo/commit/45ec3663a51a9006a38f0193a910fbab2f3ecb65))
