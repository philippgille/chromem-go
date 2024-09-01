Changelog
=========

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

vNext
-----

v0.7.0 (2024-09-01)
-------------------

Highlights in this release are the possibility to export/import the DB to/from object storage like S3, a way to run a *negative* query and either filter or subtract the results from the regular query results, and the license change from AGPL to MPL. But many other additions, improvements and important fixes made it into this release as well, all without breaking changes:

### Added

- Added `DB.ExportToWriter()` to allow users to pass any `io.Writer` implementation for the DB export, not just a file. This allows for example to export the DB to AWS S3 or compatible services (like Ceph, MinIO etc.). (PR [#71](https://github.com/philippgille/chromem-go/pull/71))
- Added `DB.ImportFromReader()` to allow users to pass any `io.ReadSeeker` implementation for the DB import, not just a file. This allows for example to import the DB from AWS S3 or compatible services (like Ceph, MinIO etc.). (PR [#72](https://github.com/philippgille/chromem-go/pull/72))
- Added example code for S3 export/import with the ⬆️ new methods (PR [#73](https://github.com/philippgille/chromem-go/pull/73))
- Added Azure OpenAI compatibility (PR [#74](https://github.com/philippgille/chromem-go/pull/74) by [@iwilltry42](https://github.com/iwilltry42))
- Added lint job in GitHub Action (PR [#82](https://github.com/philippgille/chromem-go/pull/82) by [@erikdubbelboer](https://github.com/@erikdubbelboer))
- Added the feature to run a *negative* query and either filter or subtract them from the regular query results (PR [#80](https://github.com/philippgille/chromem-go/pull/80) by [@erikdubbelboer](https://github.com/@erikdubbelboer))
  - This PR also added the new `DB.QueryWithOptions` method and related options structs and constants for future extensibility without breaking the parameter list of the query method!
- Added option to only import/export selected collections to/from a DB (PR [#88](https://github.com/philippgille/chromem-go/pull/88) by [@iwilltry42](https://github.com/iwilltry42))
- Added Google / GCP Vertex AI embedding function (PR [#91](https://github.com/philippgille/chromem-go/pull/91) by [@iwilltry42](https://github.com/iwilltry42) and [#93](https://github.com/philippgille/chromem-go/pull/93))
- Added new embedding model constants for Jina and Mixedbread (PR [#94](https://github.com/philippgille/chromem-go/pull/94))
- Added `Collection.GetByID()` to get a document for a known ID (PR [#97](https://github.com/philippgille/chromem-go/pull/97), for issue [#95](https://github.com/philippgille/chromem-go/issues/95))

### Improved

- Changed license from AGPL to MPL (PR [#87](https://github.com/philippgille/chromem-go/pull/87))
- Updated `golangci-lint` in CI to its latest version (PR [#99](https://github.com/philippgille/chromem-go/pull/99))
- Added Go 1.23 to the CI build matrix (PR [#98](https://github.com/philippgille/chromem-go/pull/98))
- Improved Godoc (PR [#100](https://github.com/philippgille/chromem-go/pull/100))

### Fixed

- The `Collection.QueryEmbedding()` call assumed/expected the query embedding from the parameter to be normalized already, but it wasn't documented and it's also inconvenient for users who use an embedding model/API that doesn't return normalized embeddings. Now we check whether the embedding is normalized and if it's not then we normalize it. (PR [#77](https://github.com/philippgille/chromem-go/pull/77))
  - (Currently `chromem-go` only does cosine similarity, and document embeddings are already being normalized, so the query embedding has to be normalized as well. In the future we might offer other distance functions or allow to inject your own and make the normalization optional)
- Fixed test panic on unexpected pass (PR [#78](https://github.com/philippgille/chromem-go/pull/78))
- Fixed out of range panic on query (PR [#79](https://github.com/philippgille/chromem-go/pull/79))
- Fixed all `golangci-lint` warnings (PR [#82](https://github.com/philippgille/chromem-go/pull/82) by [@erikdubbelboer](https://github.com/@erikdubbelboer))
- Fixed grammar and inconsistent receiver names (PR [#85](https://github.com/philippgille/chromem-go/pull/85) by [@codefromthecrypt](https://github.com/codefromthecrypt))
- Fixed link in Godoc (PR [#89](https://github.com/philippgille/chromem-go/pull/89))

v0.6.0 (2024-04-25)
-------------------

Highlights in this release are an extended interface, experimental WebAssembly bindings, and the option to use a custom Ollama URL. But also the fact that three people contributed to this release! Thank you so much! 🙇‍♂️

### Added

- Added `Collection.Delete()` to delete documents from a collection (PR [#63](https://github.com/philippgille/chromem-go/pull/63) by [@iwilltry42](https://github.com/iwilltry42))
- Added an experimental WebAssembly binding (package `wasm`) and example (PR [#69](https://github.com/philippgille/chromem-go/pull/69))

### Improved

- Use prefixes for `nomic-embed-text` model in RAG-Wikipedia-Ollama example (PR [#49](https://github.com/philippgille/chromem-go/pull/49), [#65](https://github.com/philippgille/chromem-go/pull/65))
  - Thanks [@rinor](https://github.com/rinor) for pointing out the bug!
- Made Ollama URL configurable (PR [#64](https://github.com/philippgille/chromem-go/pull/64) by [@erikdubbelboer](https://github.com/@erikdubbelboer))
- Added building of code examples to CI (PR [#66](https://github.com/philippgille/chromem-go/pull/66))
- Improved RAG template (PR [#67](https://github.com/philippgille/chromem-go/pull/67))

### Breaking changes

- `NewEmbeddingFuncOllama` now requires a second parameter for the base URL. But it can be empty to use the default which was also used in the past.

### New Contributors

- [@rinor](https://github.com/rinor) made their first contribution by suggesting a fix in <https://github.com/philippgille/chromem-go/pull/49>
- [@iwilltry42](https://github.com/iwilltry42) made their first contribution in <https://github.com/philippgille/chromem-go/pull/63>
- [@erikdubbelboer](https://github.com/@erikdubbelboer) made their first contribution in <https://github.com/philippgille/chromem-go/pull/64>

**Full Changelog**: <https://github.com/philippgille/chromem-go/compare/v0.5.0...v0.6.0>

v0.5.0 (2024-03-23)
-------------------

Highlights in this release are query performance improvements (5x faster, 98% fewer memory allocations), export/import of the entire DB to/from a single file with optional gzip-compression and AES-GCM encryption, optional gzip-compression for the regular persistence, a new code example for semantic search across 5,000 arXiv papers, and an embedding func for [Cohere](https://cohere.com/models/embed).

### Added

- Added arXiv semantic search example (PR [#45](https://github.com/philippgille/chromem-go/pull/45))
- Added basic query benchmark (PR [#46](https://github.com/philippgille/chromem-go/pull/46))
- Added unit test for collection query errors (PR [#51](https://github.com/philippgille/chromem-go/pull/51))
- Added `Collection.QueryEmbedding()` method for when you already have the embedding of your query (PR [#52](https://github.com/philippgille/chromem-go/pull/52))
- Added export and import of the entire DB to/from a single file, with optional gzip-compression and AES-GCM encryption (PR [#58](https://github.com/philippgille/chromem-go/pull/58))
- Added optional gzip-compression to the regular persistence (i.e. the DB from `NewPersistentDB()` which writes a file for each added collection and document) (PR [#59](https://github.com/philippgille/chromem-go/pull/59))
- Added minimal example (PR [#60](https://github.com/philippgille/chromem-go/pull/60), [#62](https://github.com/philippgille/chromem-go/pull/62))
- Added embedding func for [Cohere](https://cohere.com/models/embed) (PR [#61](https://github.com/philippgille/chromem-go/pull/61))

### Improved

- Changed the example link target to directory instead of `main.go` file (PR [#43](https://github.com/philippgille/chromem-go/pull/43))
- Improved query performance (5x faster, 98% fewer memory allocations) (PR [#47](https://github.com/philippgille/chromem-go/pull/47), [#53](https://github.com/philippgille/chromem-go/pull/53), [#54](https://github.com/philippgille/chromem-go/pull/54))
  - <details><summary>benchstat output</summary>

    ```text
    goos: linux
    goarch: amd64
    pkg: github.com/philippgille/chromem-go
    cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz
                                        │    before     │               after                 │
                                        │    sec/op     │    sec/op     vs base               │
    Collection_Query_NoContent_100-8      413.69µ ±  4%   90.79µ ±  2%  -78.05% (p=0.002 n=6)
    Collection_Query_NoContent_1000-8     2759.4µ ±  0%   518.8µ ±  1%  -81.20% (p=0.002 n=6)
    Collection_Query_NoContent_5000-8     12.980m ±  1%   2.144m ±  1%  -83.49% (p=0.002 n=6)
    Collection_Query_NoContent_25000-8    66.559m ±  1%   9.947m ±  2%  -85.06% (p=0.002 n=6)
    Collection_Query_NoContent_100000-8   282.41m ±  3%   39.75m ±  1%  -85.92% (p=0.002 n=6)
    Collection_Query_100-8                416.75µ ±  2%   90.99µ ±  1%  -78.17% (p=0.002 n=6)
    Collection_Query_1000-8               2792.8µ ± 23%   595.2µ ± 13%  -78.69% (p=0.002 n=6)
    Collection_Query_5000-8               15.643m ±  1%   2.556m ±  1%  -83.66% (p=0.002 n=6)
    Collection_Query_25000-8               78.29m ±  1%   11.66m ±  1%  -85.11% (p=0.002 n=6)
    Collection_Query_100000-8             338.54m ±  5%   39.70m ± 12%  -88.27% (p=0.002 n=6)
    geomean                                12.97m         2.192m        -83.10%

                                        │      before      │               after                 │
                                        │       B/op       │     B/op      vs base               │
    Collection_Query_NoContent_100-8       1211.007Ki ± 0%   5.030Ki ± 0%  -99.58% (p=0.002 n=6)
    Collection_Query_NoContent_1000-8      12082.16Ki ± 0%   13.24Ki ± 0%  -99.89% (p=0.002 n=6)
    Collection_Query_NoContent_5000-8      60394.23Ki ± 0%   45.99Ki ± 0%  -99.92% (p=0.002 n=6)
    Collection_Query_NoContent_25000-8     301962.1Ki ± 0%   206.7Ki ± 0%  -99.93% (p=0.002 n=6)
    Collection_Query_NoContent_100000-8   1207818.1Ki ± 0%   791.4Ki ± 0%  -99.93% (p=0.002 n=6)
    Collection_Query_100-8                 1211.006Ki ± 0%   5.033Ki ± 0%  -99.58% (p=0.002 n=6)
    Collection_Query_1000-8                12082.11Ki ± 0%   13.25Ki ± 0%  -99.89% (p=0.002 n=6)
    Collection_Query_5000-8                60394.10Ki ± 0%   46.04Ki ± 0%  -99.92% (p=0.002 n=6)
    Collection_Query_25000-8               301962.1Ki ± 0%   206.8Ki ± 0%  -99.93% (p=0.002 n=6)
    Collection_Query_100000-8             1207818.1Ki ± 0%   791.4Ki ± 0%  -99.93% (p=0.002 n=6)
    geomean                                   49.13Mi        54.97Ki       -99.89%

                                        │    before     │              after                │
                                        │   allocs/op   │ allocs/op   vs base               │
    Collection_Query_NoContent_100-8        238.00 ± 0%   94.00 ± 1%  -60.50% (p=0.002 n=6)
    Collection_Query_NoContent_1000-8       2038.5 ± 0%   140.5 ± 0%  -93.11% (p=0.002 n=6)
    Collection_Query_NoContent_5000-8      10039.0 ± 0%   172.0 ± 1%  -98.29% (p=0.002 n=6)
    Collection_Query_NoContent_25000-8     50038.0 ± 0%   204.0 ± 1%  -99.59% (p=0.002 n=6)
    Collection_Query_NoContent_100000-8   200038.0 ± 0%   232.0 ± 3%  -99.88% (p=0.002 n=6)
    Collection_Query_100-8                  238.00 ± 0%   94.50 ± 1%  -60.29% (p=0.002 n=6)
    Collection_Query_1000-8                 2038.0 ± 0%   141.0 ± 1%  -93.08% (p=0.002 n=6)
    Collection_Query_5000-8                10038.0 ± 0%   174.5 ± 2%  -98.26% (p=0.002 n=6)
    Collection_Query_25000-8               50038.0 ± 0%   205.5 ± 2%  -99.59% (p=0.002 n=6)
    Collection_Query_100000-8             200038.5 ± 0%   233.0 ± 1%  -99.88% (p=0.002 n=6)
    geomean                                 8.661k        161.4       -98.14%
    ```

    </details>
- Extended parameter validation (PR [#50](https://github.com/philippgille/chromem-go/pull/50), [#51](https://github.com/philippgille/chromem-go/pull/51))
- Simplified unit tests (PR [#55](https://github.com/philippgille/chromem-go/pull/55))
- Improve `NewPersistentDB()` path handling (PR [#56](https://github.com/philippgille/chromem-go/pull/56))
- Improve loading of persistent DB (PR [#57](https://github.com/philippgille/chromem-go/pull/57))
- Increased unit test coverage in various of the other listed PRs

### Fixed

- Fixed path joining (PR [#44](https://github.com/philippgille/chromem-go/pull/44))

### Breaking changes

- Due to vectors now being normalized at the time of adding the document to the collection instead of when querying, the persisted data from prior versions is incompatible with this version (PR [#47](https://github.com/philippgille/chromem-go/pull/47))

v0.4.0 (2024-03-06)
-------------------

Highlights in this release are optional persistence, an extended interface, support for creating embeddings with [Ollama](https://github.com/ollama/ollama/), the exporting of the `Document` struct, and more Go-idiomatic methods to add documents to collections.

### Added

- Extended the interface:
  - `DB.ListCollections()` (PR [#12](https://github.com/philippgille/chromem-go/pull/12))
  - `DB.GetCollection()` (PR [#13](https://github.com/philippgille/chromem-go/pull/13) + [#19](https://github.com/philippgille/chromem-go/pull/19))
  - `DB.DeleteCollection()` (PR [#14](https://github.com/philippgille/chromem-go/pull/14))
  - `DB.Reset()` (PR [#15](https://github.com/philippgille/chromem-go/pull/15))
  - `DB.GetOrCreateCollection()` (PR [#22](https://github.com/philippgille/chromem-go/pull/22))
  - `Collection.Count()` (PR [#27](https://github.com/philippgille/chromem-go/pull/27))
  - `Document` struct, `NewDocument()` function, `Collection.AddDocument()` and `Collection.AddDocuments()` methods (PR [#34](https://github.com/philippgille/chromem-go/pull/34))
    - More Go-idiomatic alternatives to `Collection.Add()`
- Added various unit tests (PR [#20](https://github.com/philippgille/chromem-go/pull/20), [#39](https://github.com/philippgille/chromem-go/pull/39))
- Added optional persistence! Via multiple PRs:
  - Write in PR [#23](https://github.com/philippgille/chromem-go/pull/23), [#24](https://github.com/philippgille/chromem-go/pull/24), [#31](https://github.com/philippgille/chromem-go/pull/31)
  - Read in PR [#25](https://github.com/philippgille/chromem-go/pull/25)
  - Delete in PR [#26](https://github.com/philippgille/chromem-go/pull/26)
- Added support for creating embeddings with [Ollama](https://github.com/ollama/ollama/) (PR [#32](https://github.com/philippgille/chromem-go/pull/32))
- Added example documentation (PR [#42](https://github.com/philippgille/chromem-go/pull/42))

### Improved

- Improved example (PR [#11](https://github.com/philippgille/chromem-go/pull/11), [#28](https://github.com/philippgille/chromem-go/pull/28), [#33](https://github.com/philippgille/chromem-go/pull/33))
- Stop exporting `Collection.Metadata` (PR [#16](https://github.com/philippgille/chromem-go/pull/16))
  - Goal: Prevent direct modifications which could cause data races in case of the user doing a modification while `chromem-go` for example ranges over it during a `Collection.Query()` call.
- Copy metadata in constructors (PR [#17](https://github.com/philippgille/chromem-go/pull/17))
  - Goal: Prevent direct modifications which could cause data races in case of the user doing a modification while `chromem-go` for example ranges over it.
- Improved CI (PR [#18](https://github.com/philippgille/chromem-go/pull/18))
  - Add Go 1.22 to test matrix, update used GitHub Action from v4 to v5, use race detector during tests
- Reorganize code internally (PR [#21](https://github.com/philippgille/chromem-go/pull/21))
- Switched to newer recommended check for file related `ErrNotExist` errors (PR [#29](https://github.com/philippgille/chromem-go/pull/29))
- Added more validations in several existing methods (PR [#30](https://github.com/philippgille/chromem-go/pull/30))
- Internal variable renamed (PR [#37](https://github.com/philippgille/chromem-go/pull/37))
- Fail unit tests immediately (PR [#40](https://github.com/philippgille/chromem-go/pull/40))

### Fixed

- Fixed metadatas validation in `Collection.AddConcurrently()` (PR [#35](https://github.com/philippgille/chromem-go/pull/35))
- Fixed Godoc of `Collection.Query()` method (PR [#36](https://github.com/philippgille/chromem-go/pull/36))
- Fixed length of result slice (PR [#38](https://github.com/philippgille/chromem-go/pull/38))
- Fixed filter test (PR [#41](https://github.com/philippgille/chromem-go/pull/41))

### Breaking changes

- Because functions can't be (de-)serialized, `GetCollection` requires a new parameter of type `EmbeddingFunc`, in order to set the correct func when using a DB with persistence and it just loaded the collections and documents from storage. (PR [#25](https://github.com/philippgille/chromem-go/pull/25))
- Some methods now return an error (due to file operations when persistence is used)
- Additional validations will return an early error, but most (if not all) prior calls with the invalid parameters probably lead to some errors down the line anyway
- `Collection.Metadata` is not exported anymore
- `Result.Document` field was renamed to `Result.Content`, to avoid confusion with the now exported `Document` struct

v0.3.0 (2024-02-10)
-------------------

### Added

- Added support for more OpenAI embedding models (PR [#6](https://github.com/philippgille/chromem-go/pull/6))
- Added support for more embedding creators/providers: (PR [#10](https://github.com/philippgille/chromem-go/pull/10))
  - [Mistral](https://docs.mistral.ai/platform/endpoints/#embedding-models), [Jina](https://jina.ai/embeddings), [mixedbread.ai](https://www.mixedbread.ai/), [LocalAI](https://github.com/mudler/LocalAI)

### Improved

- Improve concurrency when adding documents to collection (PR [#2](https://github.com/philippgille/chromem-go/pull/2))
- Rename `Client` to `DB` to better indicate that the database is embedded and there's no client-server separation (PR [#3](https://github.com/philippgille/chromem-go/pull/3))
- Change OpenAPI embedding model from "text-embedding-ada-002" to "text-embedding-3-small" (PR [#4](https://github.com/philippgille/chromem-go/pull/4))
- Allow custom base URL for OpenAI, enabling the use of [Azure OpenAI](https://azure.microsoft.com/en-us/products/ai-services/openai-service), [LiteLLM](https://github.com/BerriAI/litellm), [ollama](https://github.com/ollama/ollama/blob/main/docs/openai.md) etc. (PR [#7](https://github.com/philippgille/chromem-go/pull/7))
- Renamed `EmbeddingFunc` constructors to follow best practice (PR [#9](https://github.com/philippgille/chromem-go/pull/9))

### Fixed

- Don't allow `nResults` arg < 0 (PR [#5](https://github.com/philippgille/chromem-go/pull/5))

### Breaking changes

- Several function names and signatures were changed in this release. This can happen as long as the version is at `v0.x.y`.

v0.2.0 (2024-01-01)
-------------------

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
- Removed dependency on third party library for OpenAI ([commit](https://github.com/philippgille/chromem-go/commits/1a28e1b89808cb223d67808a8e90bb9c36d2d801))
- Parallelized document querying (PR [#1](https://github.com/philippgille/chromem-go/pull/1))

v0.1.0 (2023-12-29)
-------------------

Initial release with a minimal Chroma-like interface and a working retrieval augmented generation (RAG) example.
