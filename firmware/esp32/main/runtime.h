#pragma once
#include "channel.h"
#include "provision.h"
#include "panel75.h"
typedef struct {
    ep_flash flash;
    ep_screen screen;
    ep_owner owner;
    ep_panel75 panel;
    ep_channel usb, network;
    bool boot_safe, network_fenced;
} ep_runtime;
void ep_runtime_run(ep_runtime *);
