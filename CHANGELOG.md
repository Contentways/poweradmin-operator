# Changelog

## [v1.0.3](https://github.com/Contentways/poweradmin-operator/releases/tag/v1.0.3)

### Bug Fixes

- correct zoneID json tag in DNSRecord status

## [v1.0.2](https://github.com/Contentways/poweradmin-operator/releases/tag/v1.0.2)

### Bug Fixes

- correct zoneID and recordID field names in status

## [v1.0.1](https://github.com/Contentways/poweradmin-operator/releases/tag/v1.0.1)

### Bug Fixes

- update chart image to ghcr.io, fix CRD field names, add Chart.yaml to releaser-pleaser

## [v0.1.0](https://github.com/Contentways/poweradmin-operator/releases/tag/v0.1.0)

### Features

- add nameservers field to DNSZone spec
- add Helm chart for poweradmin-operator
- add DNSRecord CRD and controller
- initial poweradmin-operator scaffold with DNSZone controller

### Bug Fixes

- rename ZoneId/RecordId to ZoneID/RecordID, exclude dot-imports from linter
- update tests for nameservers field and NewTestClient removal
- update tests for nameservers field and SDK changes
- update helm chart with nameservers support
- Helmchart
- Dockerfile
