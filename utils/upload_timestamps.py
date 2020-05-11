#!/usr/bin/env python3

# Copyright © 2020 Pavel Tisnovsky
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Script to retrieve timestamp of all objects stored in AWS S3 bucket and export them to CSV."""

import boto3
import botocore
import csv
import sys
from argparse import ArgumentParser


def connect_to_s3(aws_access_key_id, aws_secret_access_key, region_name):
    """Try to connect into AWS S3 and initialize new session."""
    session = boto3.session.Session(aws_access_key_id=aws_access_key_id,
                                    aws_secret_access_key=aws_secret_access_key,
                                    region_name=region_name)
    use_ssl = True
    endpoint_url = None
    return session.resource('s3',
                            config=botocore.client.Config(signature_version='s3v4'),
                            use_ssl=use_ssl, endpoint_url=endpoint_url)


def get_list_of_timestamps(s3_session, bucket_name, max_records=None):
    """Get a list of timestamps for all objects in selected S3 bucket."""
    bucket = s3_session.Bucket(bucket_name)

    n = 0
    timestamps = []

    for obj in bucket.objects.all():
        timestamps.append(obj.last_modified)

        # it is possible to limit number of records
        if max_records is not None:
            n += 1
            if n >= max_records:
                break

    return timestamps


def export_timestamps_into_csv(csv_file_name, timestamps):
    """Export timestamps into CSV file."""
    with open(csv_file_name, 'w') as csvfile:
        writer = csv.writer(csvfile, quoting=csv.QUOTE_NONNUMERIC)
        writer.writerow(["Timestamp"])

        for timestamp in timestamps:
            writer.writerow([timestamp])


def cli_arguments():
    """Retrieve all CLI arguments."""
    parser = ArgumentParser()
    parser.add_argument("-k", "--access_key", help="AWS access key ID", required=True)
    parser.add_argument("-s", "--secret_key", help="AWS secret access key", required=True)
    parser.add_argument("-r", "--region", help="AWS region, us-east-1 by default",
                        default="us-east-1")
    parser.add_argument("-b", "--bucket",
                        help="bucket name, insights-buck-it-openshift by default",
                        default="insights-buck-it-openshift")
    parser.add_argument("-o", "--output", help="output file name", required=True)
    parser.add_argument("-m", "--max_records", help="max records to export (default=all)",
                        default=None, type=int)

    return parser.parse_args()


def main():
    """Entry point to this script."""
    args = cli_arguments()

    # initialize s3 session and read all timestamps
    s3 = connect_to_s3(args.access_key, args.secret_key, args.region)
    timestamps = get_list_of_timestamps(s3, args.bucket, args.max_records)

    # timestamps are usually not sorted
    timestamps.sort()

    # export all timestamps into specified file
    export_timestamps_into_csv(args.output, timestamps)


if __name__ == "__main__":
    main()
