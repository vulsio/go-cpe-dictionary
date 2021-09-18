import argparse
import logging
from typing import Tuple
from deepdiff import DeepDiff
import requests
from requests.adapters import HTTPAdapter
from urllib3.util import Retry
from urllib.parse import quote
import pprint
from concurrent.futures import ThreadPoolExecutor
import os
import random
import math
import json
import shutil
import time


def diff_response(args: Tuple[str, str]):
    # Endpoint
    # /cpes/:vendor/:product
    path = f'cpes/{args[0]}/{args[1]}'

    session = requests.Session()
    retries = Retry(total=5,
                    backoff_factor=1,
                    status_forcelist=[503, 504])
    session.mount("http://", HTTPAdapter(max_retries=retries))

    try:
        response_old = requests.get(
            f'http://127.0.0.1:1325/{path}', timeout=(2.0, 30.0)).json()
        response_new = requests.get(
            f'http://127.0.0.1:1326/{path}', timeout=(2.0, 30.0)).json()
    except requests.exceptions.ConnectionError as e:
        logger.error(
            f'Failed to Connection..., err: {e}, {pprint.pformat({"args": args, "path": path}, indent=2)}')
        exit(1)
    except requests.exceptions.ReadTimeout as e:
        logger.warning(
            f'Failed to Read Response..., err: {e}, {pprint.pformat({"args": args, "path": path}, indent=2)}')
    except Exception as e:
        logger.error(
            f'Failed to GET request..., err: {e}, {pprint.pformat({"args": args, "path": path}, indent=2)}')
        exit(1)

    diff = DeepDiff(response_old, response_new, ignore_order=True)
    if diff != {}:
        logger.warning(
            f'There is a difference between old and new(or RDB and Redis):\n {pprint.pformat({"args": args, "path": path}, indent=2)}')

        diff_path = f'integration/diff/cpes/{args[0]}#{args[1]}'
        with open(f'{diff_path}.old', 'w') as w:
            w.write(json.dumps(response_old, indent=4))
        with open(f'{diff_path}.new', 'w') as w:
            w.write(json.dumps(response_new, indent=4))


parser = argparse.ArgumentParser()
parser.add_argument('mode', choices=['cpes'],
                    help='Specify the mode to test.')
parser.add_argument("--sample_rate", type=float, default=0.001,
                    help="Adjust the rate of data used for testing (len(test_data) * sample_rate)")
parser.add_argument(
    '--debug', action=argparse.BooleanOptionalAction, help='print debug message')
args = parser.parse_args()

logger = logging.getLogger(__name__)
stream_handler = logging.StreamHandler()

if args.debug:
    logger.setLevel(logging.DEBUG)
    stream_handler.setLevel(logging.DEBUG)
else:
    logger.setLevel(logging.INFO)
    stream_handler.setLevel(logging.INFO)

formatter = logging.Formatter(
    '%(levelname)s[%(asctime)s] %(message)s', "%m-%d|%H:%M:%S")
stream_handler.setFormatter(formatter)
logger.addHandler(stream_handler)

logger.info(
    f'start server mode test(mode: {args.mode})')

logger.info('check the communication with the server')
for i in range(5):
    try:
        if requests.get('http://127.0.0.1:1325/health').status_code == requests.codes.ok and requests.get('http://127.0.0.1:1326/health').status_code == requests.codes.ok:
            logger.info('communication with the server has been confirmed')
            break
    except Exception:
        pass
    time.sleep(1)
else:
    logger.error('Failed to communicate with server')
    exit(1)

list_path = f"integration/cpe.txt"
if not os.path.isfile(list_path):
    logger.error(f'Failed to find list path..., list_path: {list_path}')
    exit(1)

diff_path = f'integration/diff/{args.mode}'
if os.path.exists(diff_path):
    shutil.rmtree(diff_path)
os.makedirs(diff_path, exist_ok=True)

with open(list_path) as f:
    list = [s.strip().split("|", 1) for s in f.readlines()]
    list = random.sample(list, math.ceil(len(list) * args.sample_rate))
    with ThreadPoolExecutor() as executor:
        ins = ((e[0], e[1]) for e in list)
        executor.map(diff_response, ins)
