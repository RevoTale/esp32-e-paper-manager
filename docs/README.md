# Documentation

Migration status: original technical material is preserved, but historical
commands can still name the pico-sandbox paths. Use the root Makefile for
current independent build/test commands. Link/path normalization is ongoing.

## Practical guides

- [Container deployment](container-deployment.md)
- [ESP32 receiver and provisioning](../firmware/esp32/README.md)
- [Current screen API](screen-api.md)
- [Container acceptance evidence](container-acceptance.md)
- [Release procedure and boundaries](releasing.md)
- [Runtime dependency notice audit](runtime-dependency-notices.md)

## Reference

- [HTML/CSS profile](manager-html-css-profile-v2.md)
- [Refresh priority](refresh-priority.md)
- [Hardware sources](history/docs/epaper-hardware-sources.md)
- [Code reuse research](history/docs/epaper-code-reuse-research.md)

## Decisions and history

- [Architecture decisions](../decisions/)
- [Hardware debugging history](history/docs/epaper-debugging-history.md)
- [Original requirements and specifications](history/remote-epaper/)
- [ESP32 research](history/esp32/)
- [Historical Blitz probe](history/remote-epaper/blitz-probe/README.md)
- [Separate Pi5 service](history/remote-epaper/deploy/pi5/README.md)
- [Alternate 1.54-inch panel sources](history/remote-epaper/panel154b/SOURCES.md)

Historical documents preserve failed experiments and superseded assumptions;
they are not permission to apply another panel's electrical configuration.
Archived code/build paths refer to the original pico-sandbox checkpoint in the
migration manifest. The Pi5 and Blitz implementations are not this ESP32 product.
In particular, [the old USB manager guide](screen-manager-usb.md) describes the
superseded Blitz/EPS1 implementation, not the current launch procedure.
