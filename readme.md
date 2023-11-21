# Blender NavCap

Hack for Blender 3D to use capslock as Maya navigation. Leaves all other Blender shortcuts intact.

## Install as service

Edit blender_navcap.service and edit the two lines to edit the environment variables:

```
Environment="BLENDER_NAVCAP_MOUSE=/dev/input/event3"
Environment="BLENDER_NAVCAP_KEYBOARD=/dev/input/event5"
```

To get a list of input devices you can use `evtest`.

After that; execute `install.sh`

```
$ # check status
$ sudo systemctl status blender_navcap.service
$ # check output
$ sudo journalctl -eu blender_navcap.service 
```