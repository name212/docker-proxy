include $(CURDIR)/makefile-go/include.mk.inc

export PROJECT_NAME=docker-proxy
export GO_TARGET=./cmd

TEST_TMP_DIR = $(CURDIR)/.tmp

define BUILD_VARIABLES_TEST_ENABLE_SKIP_PERM
github.com/name212/docker-proxy/pkg/utils/permissions.AllowSkipEnv=true
endef

.tmp:
	@mkdir -p "$(TEST_TMP_DIR)"

test/build: export GO_BUILD_VARIABLES = ${BUILD_VARIABLES_TEST_ENABLE_SKIP_PERM}
test/build: go/build/current ## Build test proxy binary

test/run/proxy: export SKIP_CHECK_PERMISSIONS = true
test/run/proxy: test/clean clean/build test/build .tmp ## Run test proxy locally
	@${INCLUDE_BUILD_OUT_NAME} \
	bin_name=""; \
	if ! bin_name="$$(build_out_name)"; then \
		exit 1; \
	fi; \
	"$(BUILD_PATH)/$${bin_name}" -c "$(CURDIR)/.test.conf.yaml"

test/run/docker: check/installed/docker ## Run docker cli cmd via proxy
	@##~ RUN_CMD=cmd - command to run can be passed without 'docker'
	@##~ PROXY_USER=NAME - name of user to run if passed will find in usersConfigPath
	@##~ PROXY_TOKEN=token - token to run
	@##~ PROXY_ADDRESS=ADDR - if passed use passed proxy address
	@##~                      Otherwise, get from $(CURDIR)/.test.conf.yaml:unixSocketPath or
	@##~                      $(CURDIR)/.test.conf.yaml:bindAddress
	@${INCLUDE_ECHO} \
	if ! $(MAKE) install/yq 2>/dev/null; then \
		exit_with_err "Cannot install yq"; \
	fi; \
	if [ -z "$$RUN_CMD" ]; then \
		exit_with_err "RUN_CMD with command to run not passed"; \
	fi; \
	if [ -z "$$PROXY_USER" ] && [ -z "$$PROXY_TOKEN" ]; then \
		exit_with_err "PROXY_USER or PROXY_TOKEN  to run not passed"; \
	fi; \
	addr="$$PROXY_ADDRESS"; \
	users_cfg="$(CURDIR)/.test.users.yaml"; \
	yq_r="$(YQ_BIN_FULL) -re";\
	if [ -z "$$addr" ]; then \
		app_cfg="$(CURDIR)/.test.conf.yaml"; \
		if [ ! -f "$$app_cfg" ]; then \
			exit_with_err "PROXY_ADDRESS not passed and proxy config not found"; \
		fi; \
		if ! addr="$$($$yq_r '.unixSocketPath' "$$app_cfg" 2>/dev/null)"; then \
			if ! addr="$$($$yq_r '.bindAddress' "$$app_cfg")"; then \
				exit_with_err "Cannot extract proxy address from $$app_cfg"; \
			else \
				addr="tcp://$$addr"; \
			fi; \
		else \
			addr="unix://$$addr"; \
		fi; \
		if ! users_cfg="$$($$yq_r '.usersConfigPath' "$$app_cfg")"; then \
			exit_with_err "Cannot extract users config path from $$app_cfg"; \
		fi; \
	fi; \
	if [ -z "$$addr" ]; then \
		exit_with_err "Proxy address is empty or not extracted"; \
	fi; \
	token="$$PROXY_TOKEN"; \
	if [ -z "$$token" ]; then \
		if [ -z "$$users_cfg" ]; then \
			exit_with_err "Users config path is empty or not extracted"; \
		fi; \
		if [ ! -f "$$users_cfg" ]; then \
			exit_with_err "Users config file $$users_cfg not found"; \
		fi; \
		token_query=".users[] | select(.name == \"$$PROXY_USER\") | .token"; \
		if ! token="$$($$yq_r "$$token_query" "$$users_cfg")"; then \
			exit_with_err "Token not extracted for user $$PROXY_USER from $$users_cfg"; \
		fi; \
	fi; \
	if [ -z "$$token" ]; then \
		exit_with_err "Token is empty for user $$PROXY_USER"; \
	fi; \
	if [[ "$$RUN_CMD" != docker* ]]; then \
		RUN_CMD="docker $$RUN_CMD"; \
	fi; \
	export DOCKER_HOST="$$addr"; \
	export DOCKER_CUSTOM_HEADERS="X-Auth-Token=$$token"; \
	$$RUN_CMD	

test/clean: ## Clean temp test directory
	@rm -rfv "$(TEST_TMP_DIR)"

release/build: go/build/linux/all ## Build release binaries

release/local: export RELEASE_NAME = docker-proxy-local
release/local: export ADDITIONAL_ARTIFACTS_DIR = $(CURDIR)/install
release/local: clean/release go/test release/build common/release ## Prepare local release artifact

# rm -fv /home/ubuntu/docker-proxy-linux-amd64.tar.gz && rm -fv /root/docker-proxy-linux-amd64.tar.gz && rm -rvf /root/docker-proxy && cd /root
# cp /home/ubuntu/docker-proxy-linux-amd64.tar.gz /root/ && cd /root && mkdir -p ./docker-proxy && tar -xvf ./docker-proxy-linux-amd64.tar.gz -C ./docker-proxy && cd ./docker-proxy
release/local/deploy/amd64: ## Copy linux/amd64 tar release to remote host to /home/$REMOTE_USER/
	@##~ REMOTE_USER=NAME - remote user name
	@##~ REMOTE_IP=IP - remote IP to copy
	@${INCLUDE_ECHO} \
	if [ -z "$$REMOTE_USER" ]; then \
		exit_with_err "REMOTE_USER not passed"; \
	fi; \
	if [ -z "$$REMOTE_IP" ]; then \
		exit_with_err "REMOTE_IP not passed"; \
	fi; \
	release_file="$(CURDIR)/.release/docker-proxy-local/docker-proxy-linux-amd64.tar.gz"; \
	if [ ! -f "$$release_file" ]; then \
		exit_with_err "Release file '$$release_file' not found or not file"; \
	fi; \
	scp "$$release_file" "$${REMOTE_USER}@$${REMOTE_IP}:/home/$${REMOTE_USER}/"

release/local/build-and-deploy/amd64: release/local release/local/deploy/amd64 ## Build and deploy linux/amd64 release to remote
	@##~ REMOTE_USER=NAME - remote user name
	@##~ REMOTE_IP=IP - remote IP to copy

.PHONY: test/build test/run/proxy test/run/docker test/clean release/build release/local release/local/deploy/amd64 release/local/build-and-deploy/amd64