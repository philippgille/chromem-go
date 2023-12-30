Changelog
=========

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

vNext
-----

### Added

- Added GitHub Actions config ([commit](https://github.com/philippgille/chromem-go/commits/fae84f2069ec28bbf9f4e30dca569f447d6aee6a))
- Added `CHANGELOG.md` ([commit](https://github.com/philippgille/chromem-go/commits/bb0aa24b95ed19a743b2b5aa60098077bebdea41))
- Exported embedding creation functions ([commit](https://github.com/philippgille/chromem-go/commits/9d8ce4ae88c08bc975a0ed6b180bc01dcb2a390f))
- Added `Collection.AddConcurrently` to add embeddings concurrently ([commit](https://github.com/philippgille/chromem-go/commits/50fe3b743696d0209d2e4c617633ba335870ab7d))

### Improved

- Improved example code ([commit](https://github.com/philippgille/chromem-go/commits/c6437611d2fd48c5458b1932d5df62f90501981f))
- Removed unused field in `Client` ([commit](https://github.com/philippgille/chromem-go/commits/9c8b01ad386008d09675a26b7eca9c9605af5b1c))
- Improved validation in `Query` method ([commit](https://github.com/philippgille/chromem-go/commits/0bd196ee7c36164fad123c7b21766c7444de246d))
- Added and improved Godoc ([commit](https://github.com/philippgille/chromem-go/commits/c3a4db9563efb270af5aee585a7fca54b2ab08dc))
- Improved locking around a collection's documents ([commit](https://github.com/philippgille/chromem-go/commits/cefec66912d2fc96928154a9159ca05bd52c5149))

v0.1.0 (2023-12-29)
-------------------

Initial release with a minimal Chroma-like interface and a working retrieval augmented generation (RAG) example.
