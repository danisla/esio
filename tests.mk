SHELL := /usr/local/bin/bash

include tests-init.mk

TEST_REPO_PATTERN_DAILY := test%2Fdaily%2Ftest-v1-%25Y_%25j
TEST_REPO_PATTERN_MONTHLY := test%2Fmonthly%2Ftest-v1-%25Y_%25m

# 2016-04-07T00:00:00Z
DOY_START_TS := 1459987200
# 2016-04-10T12:00:00Z
DOY_END_TS := 1460289600

# 2016-10-01T00:00:00Z
MON_START_TS := 1475280000
# 2016-12-28T00:00:00Z
MON_END_TS := 1482883200

test: test-1-day test-3-day test-1-month test-3-month

test-1-day: $(TEST_DEPS)
	curl -f -XGET "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24 | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day"

test-3-day: $(TEST_DEPS)
	curl -f -XGET "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24*3 | bc)?repo_pattern=$(TEST_REPO_PATTERN_DAILY)&resolution=day"

test-1-month: $(TEST_DEPS)
	curl -f -XGET "http://$(APP_HOST):$(APP_PORT)/$(DOY_START_TS)/$(shell echo $(DOY_START_TS) + 3600*24*7 | bc)?repo_pattern=$(TEST_REPO_PATTERN_MONTHLY)&resolution=month"

test-3-month: $(TEST_DEPS)
	curl -f -XGET "http://$(APP_HOST):$(APP_PORT)/$(MON_START_TS)/$(shell echo $(MON_START_TS) + 3600*24*70 | bc)?repo_pattern=$(TEST_REPO_PATTERN_MONTHLY)&resolution=month"

clean-test: clean-repo clean-template
