# foam

[![Latest Release](https://img.shields.io/github/release/joelzwarrington/foam.svg)](https://github.com/joelzwarrington/foam/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/joelzwarrington/foam)](https://pkg.go.dev/github.com/joelzwarrington/foam)
[![Build Status](https://github.com/joelzwarrington/foam/workflows/ci/badge.svg)](https://github.com/joelzwarrington/foam/actions)

Higher-order TUI components for [Bubble Tea][bubbletea], composed from
[Bubbles][bubbles] primitives.

## Palette

![palette/advanced demo](./examples/palette/advanced/demo.gif)

A command palette with pluggable modes — fuzzy-filter a static list,
dispatch an async search, or mix both behind different prefixes.

[Advanced](./examples/palette/advanced) (↑ demo)
[Simple](./examples/palette/simple)
[Overlay](./examples/palette/overlay)

## Built with

- [Bubble Tea][bubbletea] — the TUI framework
- [Bubbles][bubbles] — the primitive components foam builds on
- [Lip Gloss][lipgloss] — styling and layout

## License

[MIT](./LICENSE)

[bubbletea]: https://github.com/charmbracelet/bubbletea
[bubbles]: https://github.com/charmbracelet/bubbles
[lipgloss]: https://github.com/charmbracelet/lipgloss
