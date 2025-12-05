#!/usr/bin/env python
"""Setup script for boifi-recommender"""

from setuptools import setup, find_packages

setup(
    packages=find_packages(where="src"),
    package_dir={"": "src"},
)
