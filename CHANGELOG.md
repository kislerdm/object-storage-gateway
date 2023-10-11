## v0.0.6

### Added

- [BREAKING] [[#2](https://github.com/kislerdm/object-storage-gateway/issues/2)] The configuration `objectSizeBytes` was added as the 
attribute of the `Gateway`'s `Write` method. The change enables to reduce memory consumption significantly: 
execution of the use case described in the [issue](https://github.com/kislerdm/object-storage-gateway/issues/2) will require ~10MiB of RAM instead of ~1GiB.

---
**Note** that `objectSizeBytes` can be set to `-1` when the object size is unknown. However, it will result in a greedy memory allocation strategy identical to the one used in the previous releases.

---

### v0.0.5

### Fixed

- [[#3](https://github.com/kislerdm/object-storage-gateway/issues/3)] Objects can be read after over-writing for various
  cluster sizes (tested with 0-4 instances).

### Changed

- [BREAKING] Changes the Gateway interface to reflect its nature: `StorageConnectionReadFinder` instead of `StorageConnectionReader`.

### v0.0.4

### Changed

- Renamed the module to `github.com/kislerdm/object-storage-gateway`.
- [BREAKING] Changed the `Gateway` interfaces definition.
- Improved documentation.
- Enhanced testing by introducing e2e tests which also serve to demo the application's capabilities.

### v0.0.3

### Changed

- Simplified the module's architecture by removing `Config` as part of object creation's flow.

## v0.0.2

### Fixed

- The cache which maps the object ID to the storage instance ID is removed to ensure the `stateless` condition.
- The algorith assigning the storage instance to write an object is changed. It is based on the `objectID` now.

## v0.0.1

Initial release. Supports RW operations for files of up to a few Mb.
