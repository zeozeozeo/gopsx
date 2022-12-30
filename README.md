# gopsx, a PlayStation 1 emulator written in Go

# Usage

1. Get a PlayStation 1 BIOS.
2. To boot the BIOS, run `<command> -bios "BIOS_PATH_HERE"`. The default BIOS path is `SCPH1001.BIN` for now.
3. To insert a disc, specify it's path with `<command> -disc "DISC_PATH_HERE"`. It should be a `.bin` file (`.cue` files are not supported yet)
4. You can see other arguments by running `<command> -h`. To set boolean arguments, use `<command> -arg=true` or `-arg=false`

# Status

Implemented:

-   CPU
-   DMA
-   Timers
-   Basic CD-ROM implementation
-   Gamepad (still needs testing)
-   Interrupts
-   GPU (not much)
-   GTE (very simple implementation, doesn't seem to work at this moment)

## Images

You can click on an image to see how it should look like

[![Boot animation](https://cdn.discordapp.com/attachments/783966433641365504/1056906583193432094/image.png)](https://cdn.discordapp.com/attachments/783966433641365504/1058378226222760027/image.png)

[![BIOS main menu](https://cdn.discordapp.com/attachments/783966433641365504/1056906529271455804/image.png)](https://cdn.discordapp.com/attachments/783966433641365504/1058378809574948924/image.png)

[![BIOS logo](https://cdn.discordapp.com/attachments/783966433641365504/1058376979772100638/image.png)](https://cdn.discordapp.com/attachments/783966433641365504/1058378941158666371/image.png)

# Other

Default keyboard keymappings:

|  Gamepad  |    Keyboard     |
| :-------: | :-------------: |
|   Start   |    Backspace    |
|  Select   |   Right Shift   |
|  DPadUp   |    Arrow Up     |
| DPadRight |   Arrow Right   |
| DPadDown  |   Arrow Down    |
| DPadLeft  |   Arrow Left    |
|    L2     |  Keypad Divide  |
|    R2     | Keypad Multiply |
|    L1     |    Keypad 7     |
|    R1     |    Keypad 9     |
| Triangle  |    Keypad 8     |
|  Circle   |    Keypad 6     |
|   Cross   |    Keypad 2     |
|  Square   |    Keypad 4     |

You can change them in the `main.go` file, but it would be great to be able to do that from the CLI

# Thanks

Special thanks to [simias](https://github.com/simias) for writing [this amazing guide](https://github.com/simias/psx-guide) to writing a PlayStation emulator, and to the [Nocash PSX spec](https://problemkaputt.de/psx.htm).
