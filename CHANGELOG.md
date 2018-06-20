# Change history

## 2.1.00 - 2018-06-20

### Changed

- Default behavior is now to run on the current git root instead of staging (index).  To run on staging specify `-s`.
- If no configuration file is provided, the `.gogitix.yml` file in the git root will be used (if it exists).

## 2.0.0 - 2018-06-17

### Changed

- Config file is now specified with `-c <config file>` instead of as first argument. 

### Added

- Checks can now be run on arbitrary SHA ranges, such as `gogitix HEAD^^..HEAD` or `gogitix HEAD` or `gogitix HEAD^!`.

