.DEFAULT_GOAL := help
SHELL := /bin/bash
DOCKER ?= docker
export DOCKER
export VERSION

.PHONY: help up seed release

help:
	@printf '%s\n' 'make up                      Build and start local Docker services' 'make seed                    Start services and load reconciled demo data' 'make release VERSION=v0.1.0   Publish a new tag and GitHub Release from remote main'

up:
	@"$(DOCKER)" compose --env-file .env.docker up --build -d --wait

seed: up
	@bash scripts/seed-demo.sh

release:
	@bash scripts/release.sh
