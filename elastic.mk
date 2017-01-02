SHELL := /bin/bash

ELASTICSEARCH := .elasticsearch_container
KIBANA := .kibana_container

ES_TAG := 2
KIBANA_TAG := 4

$(ELASTICSEARCH):
	docker run -d \
		--name elasticsearch \
		-p 9200:9200 \
		-p 9300:9300 \
		--entrypoint bash \
		elasticsearch:$(ES_TAG) -c '\
			bin/plugin install cloud-aws ; \
			/docker-entrypoint.sh elasticsearch' >> $(ELASTICSEARCH)
		docker exec -it elasticsearch sh -c 'bin/plugin install lmenezes/elasticsearch-kopf';

$(KIBANA):
	docker run -d \
		--name kibana \
		-p 5601:5601 \
		-e ELASTICSEARCH_URL=http://elasticsearch:9200 \
		--link elasticsearch kibana:$(KIBANA_TAG) >> $(KIBANA)

start-elastic: $(ELASTICSEARCH) $(KIBANA)

wait-elastic:
	@while [[ ! `curl -sf http://localhost:9200` ]]; do sleep 5; done
	@echo "Elasticsearch is running at: http://localhost:9200"

open-elastic: wait-elastic
	open http://localhost:5601
	open http://localhost:9200/_plugin/kopf

stop-elastic:
	-cat $(KIBANA) $(ELASTICSEARCH) | xargs docker kill
	-cat $(KIBANA) $(ELASTICSEARCH) | xargs docker rm
	rm -f $(KIBANA) $(ELASTICSEARCH)
