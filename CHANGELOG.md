## Unreleased

### Fixed

- Big files can be handled now.

## 0.0.2

### Fixed 

- The cache which maps the object ID to the storage instance ID is removed to ensure the `stateless` condition.
- The algorith assigning the storage instance to write an object is changed. It is based on the `objectID` now.

## 0.0.1

Initial release. Supports RW operations for files of up to a few Mb.
