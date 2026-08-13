#!/usr/bin/env bash

# Copyright 2026
# license that can be found in the LICENSE file.

set -Eeuo pipefail

WORKING_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

if [ -z "${FORCE_DO:-}" ]; then
    FORCE_DO=""
fi

CONST_SYSTEMD_SERVICE_NAME="docker-proxy.service"

function force_enabled() {
    if [[ "$FORCE_DO" == "true" ]]; then
        return 0
    fi

    return 1
}

function echo_err() {
	echo -e "\033[0;31m${1:-}\033[0m" >&2
}

function echo_info() {
	echo -e "\033[0;32m${1:-}\033[0m" >&2
}

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

function ask_user() {
    local prompt="$1"
    local answer=""

    if force_enabled; then
        echo_info "Force do: $prompt"
        return 0
    fi

    # shellcheck disable=SC2162
    read -p "${prompt} [y/n]: " answer

    if [[ "$answer" == "y" ]]; then
        return 0
    fi

    return 1
}

function continue_with_warn_or_exit() {
    local msg="${1:-UNKNOWN_WARN}"
    if force_enabled; then
        exit_with_err "$msg"
    fi

    echo_warn "${msg}. Continue"
    return 0
}

function get_path_owner() {
    local path="${1}"
    local path_user=""
    if ! path_user="$(stat -c '%U' "$path")"; then
        exit_with_err "Cannot get owner user for '$path'"
    fi

    echo -n "$path_user"
}

function get_path_perms() {
    local path="${1}"
    local path_perms=""
    if ! path_perms="$(stat -c '%a' "$path")"; then
        exit_with_err "Cannot get permissions for '$path'"
    fi

    echo -n "$path_perms"
}

function chown_to_root() {
    local path="${1}"
    if ! chown "root:root" "$path"; then
        exit_with_err "Cannot chown to root '$path'"
    fi
}

function chmod_path() {
    local path="${1}"
    local perm="${2}"
    if ! chmod "$perm" "$path"; then
        exit_with_err "Cannot chmod to '$perm' path '$path'"
    fi
}

function change_permissions() {
    local path="$1"
    local perms_to_set="$2"
    local path_user=""
    path_user="$(get_path_owner "$path")"
    if [[ "$path_user" != "root" ]]; then
        if ask_user "Owner user of dir '$path' is '$path_user'. Do you want to chown to root?"; then
            chown_to_root "$path"
        else
            continue_with_warn_or_exit "Skip chown dir '$path'"
        fi
    fi
    local path_perms=""
    path_perms="$(get_path_perms "$path")"
    if [[ "$path_perms" != "$perms_to_set" ]]; then
        if ask_user "Permissions of dir '$path' is '$path_perms'. Do you want to chmod to $perms_to_set?"; then
            chmod_path "$path" "$perms_to_set"
        else
            continue_with_warn_or_exit "Skip chmod '$path'"
        fi
    fi
}

function pre_systemd() {
    if systemctl is-active --quiet "$CONST_SYSTEMD_SERVICE_NAME"; then
        echo_warn "Service '$CONST_SYSTEMD_SERVICE_NAME' is running. Stop for upgrade"
        if ! systemctl stop "$CONST_SYSTEMD_SERVICE_NAME"; then
            exit_with_err "Cannot stop '$CONST_SYSTEMD_SERVICE_NAME'"
        fi
    fi
}

function post_systemd() {
    if systemctl is-active --quiet "$CONST_SYSTEMD_SERVICE_NAME"; then
        echo_warn "Service '$CONST_SYSTEMD_SERVICE_NAME' is running. Stop for post-upgrade"
        if ! systemctl stop "$CONST_SYSTEMD_SERVICE_NAME"; then
            exit_with_err "Cannot stop '$CONST_SYSTEMD_SERVICE_NAME'"
        fi
    fi

    echo_info "Reload systemd daemon"

    if ! systemctl daemon-reload; then
        exit_with_err "Cannot daemon reload"
    fi

    if ! systemctl enable --now "$CONST_SYSTEMD_SERVICE_NAME"; then
        exit_with_err "Cannot enable '$CONST_SYSTEMD_SERVICE_NAME'"
    fi

    if ! systemctl is-active --quiet "$CONST_SYSTEMD_SERVICE_NAME"; then
        echo_info "Start service '$CONST_SYSTEMD_SERVICE_NAME'"
        if ! systemctl start "$CONST_SYSTEMD_SERVICE_NAME"; then
            exit_with_err "Cannot start '$CONST_SYSTEMD_SERVICE_NAME'"
        fi
    fi
}

function copy_execs() {
    echo_info "Copy docker-proxy exec..."

    local exec_src="${WORKING_DIR}/docker-proxy"
    local exec_dest="/usr/local/bin/docker-proxy"

    if [ ! -f "$exec_src" ]; then
        exit_with_err "Source docker proxy exec not found or not file"
    fi

    echo_info "Copy docker-proxy exec from '$exec_src' to '$exec_dest'"

    if ! cp "$exec_src" "$exec_dest"; then
        exit_with_err "Cannot copy docker-proxy exec from '$exec_src' to '$exec_dest'"
    fi

    change_permissions "$exec_dest" "755"

    echo_info "Copy docker-proxy-prepare-user exec..."

    local prepare_user_src="${WORKING_DIR}/docker-proxy-prepare-user.sh"
    local prepare_user_dest="/usr/local/bin/docker-proxy-prepare-user"

     if [ ! -f "$prepare_user_src" ]; then
        exit_with_err "Source docker-proxy-prepare-user exec not found or not file"
    fi

    echo_info "Copy docker-proxy-prepare-user exec from '$prepare_user_src' to '$prepare_user_dest'"

    if ! cp "$prepare_user_src" "$prepare_user_dest"; then
        exit_with_err "Cannot copy docker-proxy exec from '$exec_src' to '$exec_dest'"
    fi

    change_permissions "$prepare_user_dest" "755"
}

function prepare_files() {
    echo_info "Prepare directories..."

    local prepare_dir=""
    while IFS= read -r -d '' prepare_dir; do
        local trimmed_dir="${prepare_dir#"$WORKING_DIR"}"
        echo_info "Got dir '$trimmed_dir' to prepare"
        echo_info "Create dir '$trimmed_dir'"
        if ! ask_user "Create dir '$trimmed_dir'?"; then
            exit_with_err "Disallow create dir '$trimmed_dir'"
        fi
        if ! mkdir -p "$trimmed_dir"; then
            exit_with_err "Cannot create dir '$trimmed_dir'"
        fi

        change_permissions "$trimmed_dir" "700"
    done < <(find "$WORKING_DIR" -type d -links 2 -print0)

    echo_info "Prepare configuration files..."

    local prepare_file=""
    while IFS= read -r -d '' prepare_file; do
        local trimmed_file="${prepare_file#"$WORKING_DIR"}"
        echo_info "Got file '$prepare_file' to prepare"
        
        local should_copy="true"
        if [ -f "$trimmed_file" ]; then
            if [[ "$trimmed_file" == *.conf.yaml ]]; then
                echo_warn "Found exists proxy conf file '$trimmed_file' Skip copy"
                should_copy=""
            fi
        fi
        if [[ "$should_copy" == "true" ]]; then
            if ! ask_user "Copy '$prepare_file' to '$trimmed_file'?"; then
                exit_with_err "Disallow copy to '$trimmed_file'"
            fi
        fi
        change_permissions "$trimmed_file" "600"
    done < <(find "$WORKING_DIR" -type f -mindepth 2 -print0)

    local systemd_file="/etc/systemd/system/${CONST_SYSTEMD_SERVICE_NAME}"

    if [ ! -f "$systemd_file" ]; then
        exit_with_err "Systemd service file '$systemd_file' not found or not file. Cannot continue"
    fi
}

function main() {
    if ! ask_user "Do you prepare users file users.conf.yaml?"; then
        exit_with_err "Disallow continue without prepare user"
    fi
    
    pre_systemd
    copy_execs
    prepare_files
    post_systemd
}