package main

import(
    "fmt"
    "os"
    "os/exec"

    "./utils"
)

func main(){
    genPartion := &utils.GenPartition{
        IfName:"bl602/partition/partition_cfg_2M.toml",
        OfName:"bl602/image/partition.bin",
    }
    genPartion.CreatePartitionBin() 
    fmt.Println(genPartion)

    genBoot2Image := &utils.Image{
        IfBootInfoName:"bl602/efuse_bootheader/efuse_bootheader_cfg.conf",
        IfBinName:"bl602/builtin_imgs/blsp_boot2.bin",
        OfImageName:"bl602/image/boot2image.bin",
        FWOffset:0x2000,
    }
    genBoot2Image.CreateImage()

    genFWImage := &utils.Image{
        IfBootInfoName:"bl602/efuse_bootheader/efuse_bootheader_cfg.conf",
        IfBinName:"bl602/bl602.bin",
        OfImageName:"bl602/image/fwimage.bin",
        FWOffset:0x1000,
    }
    genFWImage.CreateImage()
    
    args :=[]string{"dts2dtb.py","bl602/device_tree/bl_factory_params_IoTKitA_40M.dts","bl602/image/ro_params.dtb"}
	cmd := exec.Command("python3", args...)
	cmd.Stdout = os.Stdout
	err := cmd.Run()
    if err != nil {
        fmt.Println(err)
    }
    bins := []string{
        "bl602/image/boot2image.bin@0x000000",
        "bl602/image/partition.bin@0xE000",
        "bl602/image/partition.bin@0xF000",
        "bl602/image/fwimage.bin@0x10000",
        "bl602/image/ro_params.dtb@0x1F8000",
    }
    utils.StartProgram("/dev/ttyUSB0", nil, 512000, "bl602/eflash_loader/eflash_loader_40m.bin", 2000000, bins, 5000)
}
