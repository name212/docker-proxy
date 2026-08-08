include $(CURDIR)/makefile-go/include.mk.inc

export PROJECT_NAME=docker-proxy
export GO_TARGET=./cmd

TEST_TMP_DIR = $(CURDIR)/.tmp

.tmp:
	@mkdir -p "$(TEST_TMP_DIR)"

test/build: go/build/current

test/run/proxy: export SKIP_CHECK_PERMISSIONS = true
test/run/proxy: test/clean clean/build test/build .tmp
	@${INCLUDE_BUILD_OUT_NAME} \
	bin_name=""; \
	if ! bin_name="$$(build_out_name)"; then \
		exit 1; \
	fi; \
	"$(BUILD_PATH)/$${bin_name}" -c "$(CURDIR)/.test.conf.yaml"

test/run/docker: check/installed/docker install/yq ## Run docker cli cmd via proxy
	@##~ RUN_CMD=cmd - command to run
	@##~ PROXY_USER=NAME - name of user to run
	@##~ PROXY_ADDRESS=ADDR - if passed use passed proxy address
	@##~                      Otherwise, get from $(CURDIR)/.test.conf.yaml:unixSocketPath or
	@##~                      $(CURDIR)/.test.conf.yaml:bindAddress
	@${INCLUDE_ECHO} \
	if [ -z "$$RUN_CMD" ]; then \
		exit_with_err "RUN_CMD with command to run not passed"; \
	fi; \
	if [ -z "$$PROXY_USER" ]; then \
		exit_with_err "PROXY_USER with user to run not passed"; \
	fi; \
	addr="$$PROXY_ADDRESS"; \
	users_cfg="$(CURDIR)/.test.users.yaml"; \
	yq_r="$(YQ_BIN_FULL) -re";\
	if [ -z "$$addr" ]; then \
		app_cfg="$(CURDIR)/.test.conf.yaml"; \
		if [ ! -f "$$app_cfg" ]; then \
			exit_with_err "PROXY_ADDRESS not passed and proxy config not found"; \
		fi; \
		if ! addr="$$($$yq_r '.unixSocketPath' "$$app_cfg")"; then \
			if ! addr="$$($$yq_r '.bindAddress' "$$app_cfg")"; then \
				exit_with_err "Cannot extract proxy address from $$app_cfg"; \
			else \
				addr="http://$$addr"; \
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
	if [ -z "$$users_cfg" ]; then \
		exit_with_err "Users config path is empty or not extracted"; \
	fi; \
	if [ ! -f "$$users_cfg" ]; then \
		exit_with_err "Users config file $$users_cfg not found"; \
	fi; \
	token_query=".users[] | select(.name == \"$$PROXY_USER\") | .token"; \
	token=""; \
	if ! token="$$($$yq_r "$$token_query" "$$users_cfg")"; then \
		exit_with_err "Token not extracted for user $$PROXY_USER from $$users_cfg"; \
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

test/clean:
	@rm -rfv "$(TEST_TMP_DIR)"