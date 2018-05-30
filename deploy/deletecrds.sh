#!/bin/bash

for crd in "$@"
do
    kubectl delete deployments "$crd"
    kubectl delete services "$crd"
    kubectl delete nfsshare "$crd"
done