# BLOpenFlasher
This is open source version for flash_tools, not perfect but open source totally. The current version also require python3 env, and if not please use tools to convert dts to dtb.

# Installing dependencies
```bash
go get gopkg.in/ini.v1 github.com/pelletier/go-toml github.com/albenik/go-serial
```

# Usage
Switch BL602 to program mode by pulling-up GPIO8 during booting, connect via the uart(which is hardcode now), and then run "go run flash_tool.go". Bin files are generated and then bl602 will be programmed. 
