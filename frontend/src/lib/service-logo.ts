import type { Component } from 'svelte';

// Marks only, never wordmarks: these render inside a 28px tile, where a logo
// with the brand name set beside it is illegible. Where the set ships both,
// the `-icon` variant is the mark.
import Couchdb from '~icons/logos/couchdb-icon';
import Django from '~icons/logos/django-icon';
import Docker from '~icons/logos/docker-icon';
import Elasticsearch from '~icons/logos/elasticsearch';
import Gitlab from '~icons/logos/gitlab-icon';
import Grafana from '~icons/logos/grafana';
import Influxdb from '~icons/logos/influxdb-icon';
import Java from '~icons/logos/java';
import Kafka from '~icons/logos/kafka-icon';
import Laravel from '~icons/logos/laravel';
import Mariadb from '~icons/logos/mariadb-icon';
import Meilisearch from '~icons/logos/meilisearch';
import Memcached from '~icons/logos/memcached';
import Metabase from '~icons/logos/metabase';
import Mongodb from '~icons/logos/mongodb-icon';
import Mysql from '~icons/logos/mysql-icon';
import Nats from '~icons/logos/nats-icon';
import Nextjs from '~icons/logos/nextjs-icon';
import Nginx from '~icons/logos/nginx';
import Nodejs from '~icons/logos/nodejs-icon';
import Opensearch from '~icons/logos/opensearch-icon';
import Php from '~icons/logos/php';
import Postgresql from '~icons/logos/postgresql';
import Prometheus from '~icons/logos/prometheus';
import Python from '~icons/logos/python';
import Rabbitmq from '~icons/logos/rabbitmq-icon';
import Redis from '~icons/logos/redis';
import Sentry from '~icons/logos/sentry-icon';
import Strapi from '~icons/logos/strapi-icon';
import Supabase from '~icons/logos/supabase-icon';
import Typesense from '~icons/logos/typesense-icon';
import Vault from '~icons/logos/vault-icon';
import Wordpress from '~icons/logos/wordpress-icon';

const LOGOS: Record<string, Component> = {
	couchdb: Couchdb,
	django: Django,
	docker: Docker,
	elasticsearch: Elasticsearch,
	gitlab: Gitlab,
	grafana: Grafana,
	influxdb: Influxdb,
	java: Java,
	openjdk: Java,
	kafka: Kafka,
	laravel: Laravel,
	mariadb: Mariadb,
	meilisearch: Meilisearch,
	memcached: Memcached,
	metabase: Metabase,
	mongo: Mongodb,
	mongodb: Mongodb,
	mysql: Mysql,
	nats: Nats,
	next: Nextjs,
	nextjs: Nextjs,
	nginx: Nginx,
	node: Nodejs,
	nodejs: Nodejs,
	opensearch: Opensearch,
	php: Php,
	postgres: Postgresql,
	postgresql: Postgresql,
	prometheus: Prometheus,
	python: Python,
	rabbitmq: Rabbitmq,
	redis: Redis,
	valkey: Redis,
	sentry: Sentry,
	strapi: Strapi,
	supabase: Supabase,
	typesense: Typesense,
	vault: Vault,
	wordpress: Wordpress
};

/**
 * `ghcr.io/acme/postgres:16` → `postgres`, `redis/redis-stack:latest` → `redis`.
 * Registry host, org and tag carry no brand, and the `-suffix` variants
 * (`redis-stack`, `postgres-alpine`) are the same product.
 */
function imageKey(image: string): string {
	const name = repoName(image);
	return LOGOS[name] ? name : name.split(/[-_.]/)[0];
}

/**
 * The bare repository name, lowercased. The tag is stripped *after* the path is
 * split, because a private registry carries a port: in
 * `registry.local:5000/team/node-api:v2` the first colon is not the tag.
 */
function repoName(image: string): string {
	const repo = image.split('@')[0];
	const last = repo.split('/').pop() ?? repo;
	return last.split(':')[0].toLowerCase();
}

/** The logo for a Docker image, or undefined — callers fall back to a monogram. */
export function serviceLogo(image: string): Component | undefined {
	return LOGOS[imageKey(image)];
}

/** `ghcr.io/acme/postgres:16` → `P`. The fallback when no logo is mapped. */
export function serviceInitial(image: string): string {
	return (repoName(image)[0] ?? '?').toUpperCase();
}
