#!/usr/bin/env bash

# Copyright 2026
# license that can be found in the LICENSE file.

set -Eeuo pipefail

if [ -z "${CONST_PROXY_APP_CONF:-}" ]; then
    CONST_PROXY_APP_CONF="/etc/docker-proxy/app.conf.yaml"
fi

function echo_err() {
	echo -e "\033[0;31m${1:-}\033[0m" >&2
}

function echo_info() {
	echo -e "\033[0;32m${1:-}\033[0m" >&2
}

# shellcheck disable=SC2329
function echo_warn() {
	echo -e "\033[0;33m${1:-}\033[0m" >&2
}

function exit_with_err() {
    local exit_code="${2:-}"

    if [ -z "$exit_code" ]; then
        exit_code="1"
    fi

    if [ "$exit_code" -eq 0 ]; then
        exit_code="1"
    fi

    echo_err "${1:-Error}"
    exit "$exit_code"
}

function usage() {
    local exit_code="${1-}"
    if [ -z "$exit_code" ]; then
      exit_code="0"
    fi
    echo "Usage: $0 [OPTIONS]"
    echo "Prepare linux user for use docker proxy"
    echo "Create .docker_bash_alias in home directory"
    echo "and write source .docker_bash_alias to .bashrc if need"
    echo "Got users from /etc/docker-proxy/users.conf.yaml"
    echo "Got proxy address from /etc/docker-proxy/app.conf.yaml"
    echo "Warning! yq should be installed or passed path via YQ_BIN env"
    echo "Options:"
    echo "  -l|--linux-user USER_NAME - name of linux user to prepare"
    echo "  -p|--proxy-user USER_NAME - name of proxy user to consume token"  
    echo "  -h|--help                 - display this help message"
    exit "$exit_code"
}

function exit_with_err_and_usage() {
    echo_err "${1:-Error}"
    usage "1"
}

function get_user_home_dir() {
    local linux_user="$1"

    local user_passwd=""
    if ! user_passwd="$(getent passwd "$linux_user")"; then
        echo_err "Cannot get passwd ent for $linux_user"
        return 1
    fi

    local user_home=""
    if ! user_home="$(cut -d: -f6 <<<"$user_passwd")"; then 
        echo_err "Cannot extract home for $linux_user"
        return 1
    fi

    echo -n "$user_home"
}

function get_user_id() {
    local linux_user="$1"

    local user_passwd=""
    if ! user_passwd="$(getent passwd "$linux_user")"; then
        echo_err "Cannot get passwd ent for $linux_user"
        return 1
    fi

    local user_id=""
    if ! user_id="$(cut -d: -f3 <<<"$user_passwd")"; then 
        echo_err "Cannot extract uid $linux_user"
        return 1
    fi

    if ! group_id="$(cut -d: -f4 <<<"$user_passwd")"; then 
        echo_err "Cannot extract gid $linux_user"
        return 1
    fi

    echo -n "${user_id}:${group_id}"
}

function extract_proxy_address() {
    local yq_r="$1"
    local app_conf="$2"

    local proxy_address=""

    if ! proxy_address="$($yq_r '.unixSocketPath' "$app_conf" 2>/dev/null)"; then
        if ! proxy_address="$($yq_r '.bindAddress' "$app_conf")"; then
            echo_err "Cannot extract proxy address from '$app_conf'"
            return 1
        else
            proxy_address="tcp://$$proxy_address"
        fi
    else
        proxy_address="unix://$$proxy_address"
    fi

    if [ -z "$proxy_address" ]; then
        echo_err "Proxy address is empty"
        return 1
    fi

    echo -n "$proxy_address"
}

function extract_user_token() {
    local yq_r="$1"
    local app_conf="$2"
    local proxy_user="$3"

    local users_cfg=""

    if ! users_cfg="$($yq_r '.usersConfigPath' "$app_conf")"; then
		echo_err "Cannot extract users config path from '$app_conf'"
        return 1
	fi
	
    if [ ! -f "$users_cfg" ]; then
        echo_err "users config '$users_cfg' not found or not file"
        return 1
    fi

    local token_query=".users[] | select(.name == \"$proxy_user\") | .token"; \
    local token=""
	if ! token="$($yq_r "$token_query" "$users_cfg")"; then
		echo_err "Token not extracted for user '$proxy_user' from '$users_cfg'"
        return 1
	fi

    echo -n "$token"
}

function write_alias_file() { 
    local user_id="$1"
    local user_home="$2"
    local proxy_address="$3"
    local token="$4"

    local content="alias docker='DOCKER_HOST=\"$proxy_address\" DOCKER_CUSTOM_HEADERS=\"X-Auth-Token=$token\" docker"

    local alias_file="${user_home}/.docker_bash_alias"

    echo "$content" > "$alias_file"

    if ! chown "$user_id" "$alias_file"; then
        echo_err "Cannot chown alias file '$alias_file' to '$user_id'"
        return 1
    fi

    if ! chmod 600 "$alias_file"; then
        echo_err "Cannot chmod to 600 alias file '$alias_file'"
        return 1
    fi

    echo -n "${alias_file}"
}

function write_use_alias_file_to_rc() {
    local alias_file="$1"
    local user_home="$2"

    if [ ! -f "$alias_file" ]; then
        echo_err "Alias file '$alias_file' not found or not file"
        return 1
    fi

    if [ ! -s "$alias_file" ]; then
        echo_err "Alias file '$alias_file' is empty"
        return 1
    fi

    local rc_file="${user_home}/.bashrc"

    if [ ! -f "$rc_file" ]; then
        echo_err "Bash rc file '$rc_file' not found or not file"
        return 1
    fi

    if grep "source $alias_file"; then
        echo_info "Alias file already write to '$rc_file'"
        return 0
    fi

        local content=""
    content=$(cat <<EOF


if [ -f "$alias_file" ]; then
    source $alias_file
fi


EOF
    )

    echo "$content" >> "$rc_file"
}


function main() {
    local yq_r="${YQ_BIN:-}"

    if [ -n "$yq_r" ]; then
        if [ ! -x "$yq_r" ]; then
            exit_with_err_and_usage "yq bin passed via YQ_BIN env not found or not executable"
        fi
    else
        if ! command -c "yq" > /dev/null; then
            exit_with_err_and_usage "yq bin is not installed or not passed via YQ_BIN env"
        fi
        yq_r="yq"
    fi
    
    yq_r="${yq_r} -re"

    local linux_user=""
    local proxy_user=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
        -l|--linux-user)
            if [[ -z "${2:-}" || "${2:-}" == -* ]]; then
                exit_with_err "Error: Argument for $1 is missing" >&2
            fi
            linux_user="${2-}"
            shift 2
            ;;
        -p|--proxy-user)
            if [[ -z "${2:-}" || "${2:-}" == -* ]]; then
                exit_with_err "Error: Argument for $1 is missing" >&2
            fi
            proxy_user="${2-}"
            shift 2
            ;;
        -h|--help)
            usage "0"
            ;;
        *)
            exit_with_err_and_usage "Unknown argument '${1}'"
            ;;
        esac
    done

    if [ -z "$linux_user" ]; then
        exit_with_err_and_usage "Linux user is not passed"
    fi

    if [ -z "$proxy_user" ]; then
        exit_with_err_and_usage "Proxy user is not passed"
    fi

    if [ ! -f "$CONST_PROXY_APP_CONF" ]; then
        exit_with_err "Proxy config '$CONST_PROXY_APP_CONF' is not found"
    fi

    local user_home=""
    if ! user_home="$(get_user_home_dir "$linux_user")"; then
        exit_with_err "Cannot extract user home for '$linux_user'"
    fi

    local user_id=""
    if ! user_id="$(get_user_id "$linux_user")"; then
        exit_with_err "Cannot extract user id for '$linux_user'"
    fi

    local proxy_address=""
    if ! proxy_address="$(extract_proxy_address "$yq_r" "$CONST_PROXY_APP_CONF")"; then
        exit_with_err "Cannot extract proxy address"
    fi

    local token=""
    if ! token="$(extract_user_token "$yq_r" "$CONST_PROXY_APP_CONF" "$proxy_user")"; then
        exit_with_err "Cannot extract token for proxy user '$proxy_user'"
    fi

    echo_info "Got: user_id='$user_id' user_home='$user_home' proxy_address='$proxy_address' token_len='${#token}'"

    local alias_file=""
    if ! alias_file="$(write_alias_file "$user_id" "$user_home" "$proxy_address" "$token")"; then
        exit_with_err "Cannot write alias file"
    fi

    echo_info "Docker alias file written to '$alias_file'"

    if ! write_use_alias_file_to_rc "$alias_file" "$user_home"; then
        exit_with_err "Cannot write use alias file '$alias_file' to bashrc"
    fi

    echo_info "Linux user '$linux_user' prepared for use docker proxy '$proxy_address' with proxy user '$proxy_user'"

    return 0
}

main "$@"
exit $?