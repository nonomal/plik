#!/bin/bash

#
## Plik - Simple bash uploading script
#

set -e


function usage() {
    cat <<EOF
Usage: plik [options] FILE [FILE...]

A lightweight bash client for Plik file sharing.

Options:
  -u URL            Server URL (default: http://127.0.0.1:8080)
  -T TOKEN          Authentication token
  -o                OneShot (files deleted after first download)
  -r                Removable (files can be deleted by anyone)
  -S                Stream (block until someone downloads)
  -t TTL            Time to live: <N>m|h|d or seconds (default: server default)
  -e                Extend TTL on each download
  -c COMMENT        Set upload comment (markdown supported)
  -L LOGIN:PASS     Protect upload with HTTP basic auth
  -a                Archive all files as tar.gz
  -s                Encrypt with auto-generated passphrase (openssl)
  -p PASSPHRASE     Encrypt with given passphrase (openssl)
  -q                Quiet mode (output URLs only)
  -d                Debug mode (verbose curl output)
  -k                Insecure mode (skip TLS certificate verification)
  -v                Show server version
  -h, --help        Show this help
EOF
}

#
## Funcs
#
green='\e[0;32m'
endColor='\e[0m'
function jsonValue() {
    sed -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"\{0,1\}\([^,\"]*\)\"\{0,1\}.*/\1/p" | head -1
}

function qecho(){
    if [ "$QUIET" == false ]; then echo "$@"; fi
}
function generatePassphrase() {
    < /dev/urandom tr -dc A-Za-z0-9 | head -c${1:-32};echo;
}
function setTtl() {
    unit="${1: -1}"
    value="${1:: -1}"
    case "$unit" in
        "m") TTL=$(( value * 60 ));;
        "h") TTL=$(( value * 3600 ));;
        "d") TTL=$(( value * 86400 ));;
        *)   TTL=$1;;
    esac
    return
}

function urlencode() {
    local LANG=C
    local string="$1"
    local length=${#string}
    local encoded=""
    local c hex

    for (( i=0 ; i<length ; i++ )); do
        c=${string:$i:1}
        case "$c" in
            [a-zA-Z0-9.~_\-/:]) encoded+="$c" ;;
            *) printf -v hex '%%%02X' "'$c"; encoded+="$hex" ;;
        esac
    done
    echo "$encoded"
}

#
## Vars
#
PLIK_TOKEN=${PLIK_TOKEN-""}
QUIET=false
SECURE=false
PASSPHRASE=""
ARCHIVE=false
ONESHOT=false
REMOVABLE=false
STREAM=false
EXTEND_TTL=false
COMMENTS=""
LOGIN=""
PASSWORD=""
DEBUG=false
INSECURE=false
TTL=0
CURL_OPTS="-s"

#
## Read ~/.plikrc file
#

PLIKRC=${PLIKRC-"$HOME/.plikrc"}
if [ ! -f "$PLIKRC" ]; then
    PLIKRC="/etc/plik/plikrc"
fi

if [ -f "$PLIKRC" ]; then
    # Environment variable takes precedence over plikrc file
    if [ "$PLIK_URL" == "" ]; then
      URL=$(grep URL $PLIKRC | grep -Po '(http[^\"]*)')
      if [ "$URL" != "" ]; then
          PLIK_URL=$URL
      fi
    fi
    TOKEN=$(grep Token $PLIKRC | sed -n 's/^.*"\(.*\)".*$/\1/p' )
    if [ "$TOKEN" != "" ]; then
        PLIK_TOKEN=$TOKEN
    fi
fi

# Default URL to local instance
PLIK_URL=${PLIK_URL-"http://127.0.0.1:8080"}

#
## Parse arguments
#

declare -a files
while [ $# -gt 0 ] ; do
    case "$1" in
        -u)                   shift ; PLIK_URL="$1"   ; shift ;;
        -T)                   shift ; PLIK_TOKEN="$1" ; shift ;;
        -o) ONESHOT=true    ; shift ;;
        -r) REMOVABLE=true  ; shift ;;
        -S) STREAM=true     ; shift ;;
        -t)                   shift ; setTtl $1       ; shift ;;
        -e) EXTEND_TTL=true ; shift ;;
        -c)                   shift ; COMMENTS="$1"   ; shift ;;
        -L)                   shift
            LOGIN=$(echo "$1" | cut -d: -f1)
            PASSWORD=$(echo "$1" | cut -d: -f2-)
            shift ;;
        -a) ARCHIVE=true    ; shift ;;
        -s) SECURE=true     ; shift ;;
        -p) SECURE=true     ; shift ; PASSPHRASE="$1" ; shift ;;
        -q) QUIET=true      ; shift ;;
        -d) DEBUG=true      ; shift ;;
        -k) INSECURE=true   ; shift ;;
        -v) curl -s "${PLIK_URL}/version" ; echo ; exit 0 ;;
        -h|--help) usage    ; exit 0 ;;
        --) shift ;;
        -*) echo "bad option '$1'" >&2 ; exit 1 ;;
        *) files=("${files[@]}" "$1") ; shift ;;
    esac
done

if [ "${#files[@]}" == 0 ]; then
    echo "No files specified !" >&2
    exit 1
fi

#
## Create new upload
#

if [ "$PLIK_TOKEN" != "" ]; then
    AUTH_TOKEN_HEADER="-H \"X-PlikToken: $PLIK_TOKEN\""
fi

if [ "$DEBUG" == true ]; then
    CURL_OPTS="-v"
fi

if [ "$INSECURE" == true ]; then
    CURL_OPTS+=" -k"
fi

COMMENTS_JSON=""
if [ "$COMMENTS" != "" ]; then
    COMMENTS_JSON=", \"Comments\" : \"$COMMENTS\""
fi

LOGIN_JSON=""
if [ "$LOGIN" != "" ]; then
    LOGIN_JSON=", \"Login\" : \"$LOGIN\", \"Password\" : \"$PASSWORD\""
fi

OPTIONS="{ \"OneShot\" : $ONESHOT, \"Removable\" : $REMOVABLE, \"Stream\" : $STREAM, \"Ttl\" : $TTL, \"ExtendTTL\" : $EXTEND_TTL$COMMENTS_JSON$LOGIN_JSON }"
qecho -e "Create new upload on $PLIK_URL...\n"

CREATE_UPLOAD_CMD="curl $CURL_OPTS -X POST $AUTH_TOKEN_HEADER -d '$OPTIONS' ${PLIK_URL}/upload"
NEW_UPLOAD_RESP=$(eval $CREATE_UPLOAD_CMD)
UPLOAD_ID=$(echo "$NEW_UPLOAD_RESP" | jsonValue id)

DOWNLOAD_DOMAIN=$(echo "$NEW_UPLOAD_RESP" | jsonValue downloadDomain)
if [ "$DOWNLOAD_DOMAIN" == "" ]; then
  DOWNLOAD_DOMAIN=$PLIK_URL
fi

# Handle error
if [ "$UPLOAD_ID" == "" ]; then
    ERROR_MSG=$(echo "$NEW_UPLOAD_RESP" | jsonValue message)
    if [ "$ERROR_MSG" != "" ]; then
        echo "$ERROR_MSG" >&2
    elif [ "$NEW_UPLOAD_RESP" != "" ]; then
        echo "$NEW_UPLOAD_RESP" >&2
    fi
    exit 1
fi

UPLOAD_TOKEN=$(echo "$NEW_UPLOAD_RESP" | jsonValue uploadToken)
UPLOAD_TOKEN_HEADER="-H \"X-UploadToken: $UPLOAD_TOKEN\""

qecho -e " --> ${green}$PLIK_URL/#/?id=$UPLOAD_ID${endColor}\n"

#
## Test if we have to archive
#
for file in "${files[@]}"
do
    if [ -d "$file" ]; then
        ARCHIVE=true
        break
    fi
done

if [ "$ARCHIVE" == true ]; then

    ARCHIVE_NAME="archive.tar.gz"
    if [ "${#files[@]}" == 1 ]; then
        ARCHIVE_NAME="$(basename "${files[0]}").tar.gz"
    fi

    ARCHIVE_CMD="tar --create --gzip ${files[@]}"

    unset files
    declare -a files
    files[0]="$ARCHIVE_NAME"
fi


#
## Upload files
#

qecho -e "Uploading files...\n"

for FILE in "${files[@]}"
do
    STDIN=false

    FILENAME=$FILE
    if [[ "$FILE" == *\/* ]]; then
        FILENAME=$(basename "$FILE")
    fi

    UPLOAD_COMMAND=""
    if [ "$ARCHIVE" == true ]; then
        UPLOAD_COMMAND+="$ARCHIVE_CMD | "
        FILENAME=$ARCHIVE_NAME
        STDIN=true
    fi

    if [ "$SECURE" == true ]; then
        if [ "$PASSPHRASE" == "" ]; then
            PASSPHRASE=$(generatePassphrase)
        fi

        UPLOAD_COMMAND+="openssl aes-256-cbc -e -pass pass:$PASSPHRASE "
        if [ "$ARCHIVE" == false ]; then
            UPLOAD_COMMAND+="-in $FILE "
        fi

        UPLOAD_COMMAND+=" | "
        STDIN=true
    fi

    if [ "$STDIN" == true ]; then
        UPLOAD_COMMAND+="curl $CURL_OPTS -X POST $AUTH_TOKEN_HEADER $UPLOAD_TOKEN_HEADER -F \"file=@-;filename=$FILENAME\" $PLIK_URL/file/$UPLOAD_ID"
    else
        UPLOAD_COMMAND+="curl $CURL_OPTS -X POST $AUTH_TOKEN_HEADER $UPLOAD_TOKEN_HEADER -F \"file=@\\\"$FILE\\\";filename=\\\"$FILENAME\\\"\" $PLIK_URL/file/$UPLOAD_ID"
    fi

    FILE_RESP=$(eval $UPLOAD_COMMAND)
    FILE_ID=$(echo "$FILE_RESP" | jsonValue id)

    # Handle error
    if [ "$FILE_ID" == "" ]; then
        ERROR_MSG=$(echo "$FILE_RESP" | jsonValue message)
        if [ "$ERROR_MSG" != "" ]; then
            echo "$ERROR_MSG" >&2
        elif [ "$FILE_RESP" != "" ]; then
            echo "$FILE_RESP" >&2
        fi
        exit 1
    fi

    FILE_NAME=$(echo "$FILE_RESP" | jsonValue fileName)
    FILE_URL="$DOWNLOAD_DOMAIN/file/$UPLOAD_ID/$FILE_ID/$FILE_NAME"

    # Compute get command
    ENCODED_URL=$(urlencode "$FILE_URL")
    COMMAND="curl -s '$ENCODED_URL'"

    if [ "$SECURE" == true ]; then
        COMMAND+=" | openssl aes-256-cbc -d -pass \"pass:$PASSPHRASE\""
    fi

    if [ "$ARCHIVE" == true ]; then
        COMMAND+=" | tar zxvf -"
    else
        COMMAND+=" > '$FILE_NAME'"
    fi

    # Output
    if [ "$QUIET" == true ]; then
        echo "$ENCODED_URL"
    else
        echo "$COMMAND"
    fi
done
qecho

