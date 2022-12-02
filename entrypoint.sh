#!/bin/sh

if pidof $APP_BINARY_NAME; then
    export PATH=/proc/$(pidof $APP_BINARY_NAME)/root/usr/local/sbin:/proc/$(pidof $APP_BINARY_NAME)/root/usr/local/bin:/proc/$(pidof $APP_BINARY_NAME)/root/usr/sbin:/proc/$(pidof $APP_BINARY_NAME)/root/usr/bin:/proc/$(pidof $APP_BINARY_NAME)/root/sbin:/proc/$(pidof $APP_BINARY_NAME)/root/bin:$PATH
fi

exec "$@"
