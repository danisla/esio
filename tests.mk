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

###
### Tests for GET /{start}/{end}
###

test-%-day-offline: $(TEST_DEPS)
	$(eval EXP_RES := 404)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XGET "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24*$* | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day"))
	@if [ "$(RES)" != "$(EXP_RES)" ]; then echo "TEST ERROR: Expected status code '$(EXP_RES)' but saw: '$(RES)'" ; exit 1; fi
	@echo "PASSED: $@ "

test-%-month-offline: $(TEST_DEPS)
	$(eval EXP_RES := 404)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XGET "http://$(APP_HOST):$(APP_PORT)/$(MON_START_TS)/$(shell echo $(MON_START_TS) + 3600*24*30*$* | bc)?repo_pattern=$(TEST_REPO_PATTERN_MONTHLY)&resolution=month"))
	@if [ "$(RES)" != "$(EXP_RES)" ]; then echo "TEST ERROR: Expected status code '$(EXP_RES)' but saw: '$(RES)'" ; exit 1; fi
	@echo "PASSED: $@ "

test-invalid-resolution: $(TEST_DEPS)
	$(eval EXP_RES := 400)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XGET "http://$(APP_HOST):$(APP_PORT)/$(MON_START_TS)/$(MON_END_TS)?resolution=foo"))
	@if [ "$(RES)" != "$(EXP_RES)" ]; then echo "TEST ERROR: Expected status code '$(EXP_RES)' but saw: '$(RES)'" ; exit 1; fi
	@echo "PASSED: $@ "

test-missing-snapsnot:
	$(eval EXP_RES := 416)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XGET "http://$(APP_HOST):$(APP_PORT)/$(MON_START_TS)/$(MON_END_TS)?resolution=day&repo_pattern=test-foo%2Fdaily%2Ftest-v1-%25Y_%25j"))
	@if [ "$(RES)" != "$(EXP_RES)" ]; then echo "TEST ERROR: Expected status code '$(EXP_RES)' but saw: '$(RES)'" ; exit 1; fi
	@echo "PASSED: $@ "

test-out-of-range: $(TEST_DEPS)
	$(eval EXP_RES := 416)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XGET "http://$(APP_HOST):$(APP_PORT)/$(shell echo $(DOY_START_TS) - 3600*24*5 | bc)/$(shell echo $(DOY_START_TS) - 3600*24*4 | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day"))
	@if [ "$(RES)" != "$(EXP_RES)" ]; then echo "TEST ERROR: Expected status code '$(EXP_RES)' but saw: '$(RES)'" ; exit 1; fi
	@echo "PASSED: $@ "

clean-test: clean-repo clean-template

###
### Tests for POST /{start}/{end}
###

test-%-day-restore: $(TEST_DEPS)
	$(eval EXP_RES := 202)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XPOST "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24*$* | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day"))
	@if [ "$(RES)" != "$(EXP_RES)" ]; then echo "TEST ERROR: Expected status code '$(EXP_RES)' but saw: '$(RES)'" ; exit 1; fi
	@echo "TEST: Verifying restore is queued" ; \
	while [[ `curl --silent -XGET "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24*$* | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day" | jq '.restoring | length'` -ne $* ]]; do \
		echo "TEST: Wating for $* indices to queue"; sleep 2 ; done
	@echo "TEST: Verifying index was restored" ; \
	while [[ `curl --silent -XGET "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24*$* | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day" | jq '.ready | length'` -ne $* ]]; do \
		echo "TEST: Wating for $* indices to come online"; sleep 2; done
	@echo "PASSED: $@ "

###
### Tests for DELETE /{start}/{end}
###

test-%-day-delete: $(TEST_DEPS)
	$(eval EXP_RES := 202)
	$(eval ROUTE := "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24*$* | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day")
	@echo $(ROUTE)
	$(eval RES := $(shell curl --silent --output /dev/stderr --write-out "%{http_code}" -XDELETE $(ROUTE)))
	@if [ "$(RES)" != "$(EXP_RES)" ]; then echo "TEST ERROR: Expected status code '$(EXP_RES)' but saw: '$(RES)'" ; exit 1; fi
	@echo "TEST: Verifying delete is queued" ; \
	while [[ `curl --silent -XGET "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24*$* | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day" | jq '.deleting | length'` -ne $* ]]; do \
		echo "TEST: Wating for $* indices to queue"; sleep 2 ; done
	@echo "TEST: Verifying indices were deleted" ; \
	while [[ `curl --silent -XGET "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24*$* | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day" | jq '.pending | length'` -ne $* ]]; do \
		echo "TEST: Wating for $* indices to be deleted"; sleep 2; done
	@echo "PASSED: $@ "
