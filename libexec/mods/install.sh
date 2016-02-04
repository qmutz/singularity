#!/bin/sh
# 
# Copyright (c) 2015, Gregory M. Kurtzer
# All rights reserved.
# 
# Copyright (c) 2015, The Regents of the University of California,
# through Lawrence Berkeley National Laboratory (subject to receipt of
# any required approvals from the U.S. Dept. of Energy).
# All rights reserved.
# 
# 

# Dependency resolvers must register their own functions to these lists
ALL_RESOLVERS=" " # Run these on all files
BIN_RESOLVERS=" " # Run these on all executable files
TXT_RESOLVERS=" " # Run these on all text scripts and data files

export ALL_RESOLVERS BIN_RESOLVERS TXT_RESOLVERS


for i in $libexecdir/singularity/mods/deps/*.sh; do
    if [ -f "$i" ]; then
        . "$i"
    fi
done

message 2 "ALL_RESOLVERS='$ALL_RESOLVERS'\n";
message 2 "BIN_RESOLVERS='$BIN_RESOLVERS'\n";
message 2 "TXT_RESOLVERS='$TXT_RESOLVERS'\n";


dep_resolver() {
    if [ -f "$1" ]; then
        if file "$1" | egrep -q "ELF"; then
            for i in $BIN_RESOLVERS; do
                eval $i "$1"
            done
        fi
        if file "$1" | egrep -q "ASCII|script"; then
            for i in $TXT_RESOLVERS; do
                eval $i "$1"
            done
        fi
        for i in $ALL_RESOLVERS; do
            eval $i "$1"
        done
    fi
}



install_file() {
    file="$1"
    dir="$2"

    if [ -f "$INSTALLDIR/c/$file" ]; then
        exit 0
    fi

    if [ -e "$file" ]; then
        filename=`basename "$file"`
        if [ -z "$dir" ]; then
            dir=`dirname "$file"`
        fi
        if [ ! -d "$INSTALLDIR/c/$dir" ]; then
            mkdir -p "$INSTALLDIR/c/$dir"
        fi

        message 1 "Installing file: $dir/$filename\n"
        if ! /bin/cp -arpL "$file" "$INSTALLDIR/c/$dir/"; then
            message 0 "ERROR: Could not copy file to container: $file\n"
            exit 1
        fi
        if ! dep_resolver "$INSTALLDIR/c/$dir/$filename"; then
            message 0 "ERROR: Could not resolve dependencies for: $file\n"
            exit 1
        fi
    else
        message 0 "ERROR: File not found: $file\n"
        exit 1
    fi
}




