# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2020-04-22
### Added
- `main.go` song files size.
- `./.github/workflows/go.yml` CI that creates releases and builds an executable for it.

## [0.4.0] - 2020-04-18
### Changed
- `main.go` validate *value* from request using *prepared statement*.
- `main.go` validate *attribute* name using **song** type fields names.

### Fixed
- `main.go` a few stupid things like returning an extra empty song every time.

## [0.3.0] - 2020-03-17
### Changed
- `main.go` do scan in multiple threads. Before scan time: 20 sec. After scan time with 4 cores: 10 sec.

## [0.2.0] - 2020-03-09
### Added
- `main.go` add new fields (*Genre*, *Year*, *Format*, *FileType*) to **Song** struct.
- `main.go` add new handler to allow queries by any song's atrribute.

### Changed
- `README.md` improve a bit.

## [0.1.0] - 2020-03-08
### Added
- `main.go` first working version.
