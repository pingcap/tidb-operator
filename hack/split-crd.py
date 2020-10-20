#!/usr/bin/env python

# Copyright 2020 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

# This script is used to split our CRD YAML file to separate ones,
# with each CRD's name as its filenam.


import argparse
import logging
import os
import yaml


def read_file(filename, mode='r'):
    data = None
    with open(filename, mode) as f:
        data = f.read()
    f.close()
    return data


# write data to file
def write_file(filename, data, mode='w'):
    with open(filename, mode) as f:
        logging.debug("Writing %s of data to %s" % (len(data), filename))
        try:
            f.write(str(data, 'utf-8'))
        except TypeError:
            f.write(data)
    f.close()


# create target directory, do nothing if it already exists
def create_dir(path):
    try:
        os.makedirs(path)
        return path
    except OSError as e:
        # There is FileExistsError (divided from OSError) in Python 3.3+,
        # but only OSError in Python 2, so we use errno to check if target
        # dir already exists.
        import errno
        if e.errno == errno.EEXIST and os.path.isdir(path):
            logging.info("Target path \"%s\" already exist" % path)
            return path
        else:
            logging.fatal("Can not create dir, error is: %s" % str(e))
            exit(e.errno)
    return None


# Increase indent level for array items to fit OperatorHub's format
# Modified from: https://web.archive.org/web/20170903201521/https://pyyaml.org/ticket/64#comment:5
class MyDumper(yaml.Dumper):
    def increase_indent(self, flow=False, indentless=False):
        return super(MyDumper, self).increase_indent(flow, False)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Split CRD YAML to separate files.")
    parser.add_argument("-i", "--input", action="store",
                        required=True, help="The path to CRD file.")
    parser.add_argument("-o", "--output", action="store",
                        required=True, help="The path to output directory.")
    parser.add_argument("-v", "--verbose", action="store_true",
                        default=False, help="Print verbose output.")
    args = parser.parse_args()

    # init logger
    if args.verbose:
        logging.basicConfig(
            format='[%(levelname)s] %(message)s (at %(filename)s:%(lineno)d in %(funcName)s).',
            level=logging.DEBUG)
        logging.info("Debug logging enabled")
        logging.debug("Input arguments are: %s" % args)
    else:
        logging.basicConfig(
            format='[%(levelname)s] %(message)s.', level=logging.INFO)
        logging.info("Using logging level: INFO")

    # prepare output dir
    create_dir(args.output)

    # read CRD file
    try:
        combined_crd = yaml.load_all(
            read_file(args.input), Loader=yaml.FullLoader)
    except Exception as e:
        logging.fatal("error reading CRD file: %s" % str(e))
        exit(1)

    for crd in combined_crd:
        try:
            _name = crd['metadata']['name']
            logging.info("Read CRD %s" % _name)
        except KeyError:
            logging.error("Error parsing CRD, ignore it: %s" % str(e))
            continue

        # write the CRD to file
        try:
            _crd = yaml.dump(crd, indent=2, Dumper=MyDumper, default_flow_style=False)
        except Exception as e:
            logging.fatal("error reformating the CRD %s: %s" % (_name, str(e)))
            exit(1)

        try:
            _path = os.path.join(args.output, "%s.crd.yaml" % _name)
            write_file(_path, _crd)
            logging.info("Write CRD %s to file %s" % (_name, _path))
        except Exception as e:
            logging.error("Error writing CRD, ignore it: %s" % str(e))
            continue

    logging.info("Done")
