SHELL := /usr/local/bin/bash

BUCKET ?= disla-ss-test
BASE_PATH ?= snapshot

AWS_DEFAULT_REGION ?= us-gov-west-1
AWS_ACCESS_KEY_ID ?= REDACTED
AWS_SECRET_ACCESS_KEY ?= REDACTED
S3_HOST ?= s3-fips-us-gov-west-1.amazonaws.com
TEST_REPO ?= test

ES_DEPS := start-elastic wait-elastic

TEST_INDICES_DOY := test-v1-2016_098 test-v1-2016_099 test-v1-2016_100 test-v1-2016_101
TEST_INDICES_MONTH := test-v1-2016_09 test-v1-2016_10 test-v1-2016_11 test-v1-2016_12
TEST_INDICES_YEAR := test-v1-2013 test-v1-2014 test-v1-2015 test-v1-2016

TEST_INDICES := $(TEST_INDICES_DOY) $(TEST_INDICES_MONTH) $(TEST_INDICES_YEAR)

TEST_DEPS := $(ES_DEPS) start-server wait-server repo-$(TEST_REPO)

repo-%: $(ES_DEPS)
	@if curl -sf http://localhost:9200/_snapshot/$* > /dev/null; then echo "Snapshot repo exists: $*"; exit 0; fi ; \
	echo "Creating snapshot repo: $*"; \
	curl -XPUT http://localhost:9200/_snapshot/$* -d '{ \
	  "type": "s3", \
	  "settings": { \
	    "compress": "true", \
	    "bucket": "$(BUCKET)", \
	    "region": "$(AWS_DEFAULT_REGION)", \
	    "endpoint": "$(S3_HOST)", \
	    "base_path": "snapshot", \
	    "access_key": "$(AWS_ACCESS_KEY_ID)", \
	    "secret_key": "$(AWS_SECRET_ACCESS_KEY)" \
	  } \
	}'

template-%: $(ES_DEPS)
	@if curl -sf http://localhost:9200/_template/$* > /dev/null; then echo "Index template exists: $*"; exit 0; fi ; \
	echo "Creating index template: $*"; \
	curl -f -XPUT http://localhost:9200/_template/$* -d '{\
		"template": "test-v*-*",\
		"settings": {\
			"number_of_shards": 1,\
			"number_of_replicas": 0\
		},\
		"mappings": {\
			"events": {\
				"properties": {\
					"timestamp": {\
						"type": "date"\
					},\
					"unixtime": {\
						"type": "date"\
					},\
					"event_id": {\
						"type": "long"\
					}\
				}\
			}\
		}\
	}'

init-test-data: init-test-data-doy init-test-data-month init-test-data-year

test_data_doy.json.txt:
	echo "" > $@
	@echo "Initializing test data by DOY"; \
	EVENT_ID=1; \
	year=2016 ; \
	month=04 ; \
	declare -a DAY ; DAY[98]=07 DAY[99]=08 DAY[100]=09 DAY[101]=10 ; \
	for doy in $${!DAY[@]}; do \
		INDEX="test-v1-$${year}_$$(printf %03g $$doy)" ; \
		echo "Generating data for index: $$INDEX" ; \
		for h in `seq 0 11`; do \
			TS=$$(gdate -u --date="$${year}-$${month}-$${DAY[$$doy]}T$$h:00:00" +"%Y-%m-%dT%H:%M:%SZ"); \
			UNIX_TS=$$(gdate -u --date="$${year}-$${month}-$${DAY[$$doy]} $$h:00:00" "+%s"); \
			echo '{"update": {"_index": "'$$INDEX'", "_type": "events", "_id": "event_'$${EVENT_ID}'"}}' >> $@ ; \
			echo '{"doc_as_upsert": true, "doc": {"timestamp": "'$$TS'", "unixtime": '$$UNIX_TS'000, "event_id": '$$EVENT_ID'}}' >> $@ ; \
			((EVENT_ID=EVENT_ID+1)); \
		done; \
	done
.INTERMEDIATE: test_data_doy.json.txt

init-test-data-doy: test_data_doy.json.txt $(addprefix index-,$(TEST_INDICES_DOY))
	curl -sf -XPOST http://localhost:9200/_bulk --data-binary @$< >/dev/null
	curl -XPUT http://localhost:9200/_snapshot/$(TEST_REPO)/daily?wait_for_completion=true -d '{"indices": "'$$(echo -n $(TEST_INDICES_DOY) | tr ' ' ',')'"}'
	make $(addprefix delete-index-,$(TEST_INDICES_DOY))

test_data_month.json.txt:
	echo "" > $@
	@echo "Initializing test data by month"; \
	EVENT_ID=1; \
	year=2016 ; \
	for month in `seq -f %02g 9 12`; do \
		INDEX="test-v1-$${year}_$${month}" ; \
		echo "Generating data for index: $$INDEX" ; \
		for d in `seq 1 28`; do \
			TS=$$(gdate -u --date="$${year}-$${month}-$${d}T00:00:00" +"%Y-%m-%dT%H:%M:%SZ"); \
			UNIX_TS=$$(gdate -u --date="$${year}-$${month}-$${d} 00:00:00" "+%s"); \
			echo '{"update": {"_index": "'$$INDEX'", "_type": "events", "_id": "event_'$${EVENT_ID}'"}}' >> $@ ; \
			echo '{"doc_as_upsert": true, "doc": {"timestamp": "'$$TS'", "unixtime": '$$UNIX_TS'000, "event_id": '$$EVENT_ID'}}' >> $@ ; \
			((EVENT_ID=EVENT_ID+1)); \
		done; \
	done
.INTERMEDIATE: test_data_month.json.txt

init-test-data-month: test_data_month.json.txt $(addprefix index-,$(TEST_INDICES_MONTH)) repo-$(TEST_REPO)
	curl -sf -XPOST http://localhost:9200/_bulk --data-binary @$< >/dev/null
	curl -XPUT http://localhost:9200/_snapshot/$(TEST_REPO)/monthly?wait_for_completion=true -d '{"indices": "'$$(echo -n $(TEST_INDICES_MONTH) | tr ' ' ',')'"}' ;
	make $(addprefix delete-index-,$(TEST_INDICES_MONTH))

test_data_year.json.txt:
	echo "" > $@
	@echo "Initializing test data by year"; \
	EVENT_ID=1; \
	for year in `seq 2013 2016`; do \
		INDEX="test-v1-$${year}" ; \
		echo "Generating data for index: $$INDEX" ; \
		for month in `seq -f %02g 1 12`; do \
			for d in `seq 1 28`; do \
				TS=$$(gdate -u --date="$${year}-$${month}-$${d}T00:00:00" +"%Y-%m-%dT%H:%M:%SZ"); \
				UNIX_TS=$$(gdate -u --date="$${year}-$${month}-$${d} 00:00:00" "+%s"); \
				echo '{"update": {"_index": "'$$INDEX'", "_type": "events", "_id": "event_'$${EVENT_ID}'"}}' >> $@ ; \
				echo '{"doc_as_upsert": true, "doc": {"timestamp": "'$$TS'", "unixtime": '$$UNIX_TS'000, "event_id": '$$EVENT_ID'}}' >> $@ ; \
				((EVENT_ID=EVENT_ID+1)); \
			done; \
		done; \
	done
.INTERMEDIATE: test_data_year.json.txt

init-test-data-year: test_data_year.json.txt $(addprefix index-,$(TEST_INDICES_YEAR)) repo-$(TEST_REPO)
	curl -sf -XPOST http://localhost:9200/_bulk --data-binary @$< >/dev/null
	curl -XPUT http://localhost:9200/_snapshot/$(TEST_REPO)/yearly?wait_for_completion=true -d '{"indices": "'$$(echo -n $(TEST_INDICES_YEAR) | tr ' ' ',')'"}' ;
	make $(addprefix delete-index-,$(TEST_INDICES_YEAR))

delete-test-data: $(addprefix delete-index-,$(TEST_INDICES)) delete-snap-daily delete-snap-monthly delete-snap-yearly
	rm -f test_data_*.json.txt

index-%: template-test
	@if curl -sf http://localhost:9200/$* > /dev/null; then echo "Index exists: $*"; exit 0; fi ; \
	echo "Creating index: $*"; \
	curl -f -XPOST http://localhost:9200/$*?wait_for_active_shards=1

delete-index-%: $(ES_DEPS)
	-curl -f -XDELETE http://localhost:9200/$*

delete-snap-%: $(ES_DEPS)
	-curl -f -XDELETE http://localhost:9200/_snapshot/$(TEST_REPO)/$*?wait_for_completion=true

clean-repo:
	-curl -XDELETE http://localhost:9200/_snapshot/$(TEST_REPO)

clean-template:
	-curl -XDELETE http://localhost:9200/_template/test

ts-%:
	gdate -u --date @$*
