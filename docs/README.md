# GoHead Handbook sources

The handbook is built with [MkDocs Material](https://squidfunk.github.io/mkdocs-material/)
and published to GitHub Pages by `.github/workflows/docs.yml`.

```bash
make docs                                   # from the repository root: http://localhost:8000
mkdocs build --strict -f docs/mkdocs.yml    # what CI runs
```

- Pages: `docs/content/` · navigation: `nav` in `docs/mkdocs.yml`
- Run commands from the repository root (the changelog page includes `CHANGELOG.md`).
