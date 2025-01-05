# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.1.0] - 2025-01-05
### Added
- **User Role Management**: Introduced roles (`admin`, `reader`, `viewer`) with distinct permissions.
- **API Endpoints**: Implemented `/auth/register` and `/auth/login` routes.
- **Test Utilities**: Added helper methods for mocking JSON POST requests and initializing test servers.
- **Slug Utility**: Added a utility for generating URL-friendly slugs, including support for accented characters.

### Changed
- **Refactored Role Handling**: Centralized role fetching and validation in storage layer.
- **Improved Logging**: Enhanced request and error logging with structured logs for debugging.

### Fixed
- **Database Migrations**: Resolved issues with `AutoMigrate` not applying schema changes during tests.
- **Test Failures**: Fixed inconsistencies in test cases for duplicate entries and invalid roles.

### Legend:
- **Added**: For new features.
- **Changed**: For changes in existing functionality.
- **Fixed**: For any bug fixes.
- **Deprecated**: For soon-to-be removed features.
- **Removed**: For now-removed features.