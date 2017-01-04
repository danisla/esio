SHELL := /usr/local/bin/bash

include tests-init.mk

TEST_REPO_PATTERN_DAILY := test%2Fdaily%2Ftest-v1-%25Y_%25j
TEST_REPO_PATTERN_MONTHLY := test%2Fmonthly%2Ftest-v1-%25Y_%25m

# 2016-04-07T00:00:00Z
DOY_START_TS := 1459987200
# 2016-04-10T12:00:00Z
DOY_END_TS := 1460289600

# 2016-09-01T00:00:00Z
MON_START_TS := 1472688000
# 2016-12-28T00:00:00Z
MON_END_TS := 1482883200

test: test-invalid-resolution test-out-of-range test-1-day-offline test-3-day-offline test-1-month-offline test-3-month-offline
	@echo "All tests PASSED"

test-%-day-offline: $(TEST_DEPS)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XGET "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24*$* | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day"))
	@if [ "$(RES)" != "404" ]; then echo "ERROR: Expected status code '404' but saw: '$(RES)'" ; exit 1; fi

test-%-month-offline: $(TEST_DEPS)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XGET "http://$(APP_HOST):$(APP_PORT)/$(MON_START_TS)/$(shell echo $(MON_START_TS) + 3600*24*30*$* | bc)?repo_pattern=$(TEST_REPO_PATTERN_MONTHLY)&resolution=month"))
	@if [ "$(RES)" != "404" ]; then echo "ERROR: Expected status code '404' but saw: '$(RES)'" ; exit 1; fi

test-invalid-resolution: $(TEST_DEPS)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XGET "http://$(APP_HOST):$(APP_PORT)/$(MON_START_TS)/$(MON_END_TS)?resolution=foo"))
	@if [ "$(RES)" != "400" ]; then echo "ERROR: Expected status code '400' but saw: '$(RES)'" ; exit 1; fi

test-missing-snapsnot:
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XGET "http://$(APP_HOST):$(APP_PORT)/$(MON_START_TS)/$(MON_END_TS)?resolution=day&repo_pattern=test-foo%2Fdaily%2Ftest-v1-%25Y_%25j"))
	@if [ "$(RES)" != "416" ]; then echo "ERROR: Expected status code '416' but saw: '$(RES)'" ; exit 1; fi

test-out-of-range: $(TEST_DEPS)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XGET "http://$(APP_HOST):$(APP_PORT)/$(shell echo $(DOY_START_TS) - 3600*24*5 | bc)/$(shell echo $(DOY_START_TS) - 3600*24*4 | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day"))
	@if [ "$(RES)" != "416" ]; then echo "ERROR: Expected status code '416' but saw: '$(RES)'" ; exit 1; fi

clean-test: clean-repo clean-template
