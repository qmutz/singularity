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


BIN_RESOLVERS="ldd_resolver $BIN_RESOLVERS"



ldd_resolver() {
    for file in $@; do
        for dep in `ldd $file | sed -e 's@ @\n@g'`; do
            if [ -f "$dep" ]; then
                message 3 "found dependency:      $dep\n"
                if [ ! -f "$INSTALLDIR/c/$dep" ]; then
                    install_file "$dep"
                fi
            fi
        done
    done
}


