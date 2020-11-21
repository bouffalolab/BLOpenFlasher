# -*- coding:utf-8 -*-

import fdt
import argparse


def dts2dtb(args):
    file_dts = args.dts
    file_dtb = args.dtb
    
    with open(file_dts, "r", encoding='utf-8') as f:
        tmp_dts = f.read()
    
    tmp_dtb = fdt.parse_dts(tmp_dts)
    
    with open(file_dtb, "wb") as f:
        f.write(tmp_dtb.to_dtb(version=17))

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='dts2dtb')
    parser.add_argument('dts')
    parser.add_argument('dtb')
    args = parser.parse_args()
    parser.set_defaults(func=dts2dtb)
    args = parser.parse_args()
    args.func(args)
    