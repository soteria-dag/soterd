# Using an external Cuckoo solver

In addition to Qitmeer's cuckaroo solver (the current default), soterd has support for [John Tromp's cuckoo solvers](https://github.com/tromp/cuckoo).

Currently you can use the lean cuckoo solver with the `--lean` cli flag, or the mean gpu cuckoo solver with the `--gpu` cli flag. In order to build 

## Warning about dag compatibility

The solvers don't use the same parameters for generating cycle nonces, so if you switch between them, blocks generated with a different solver won't verify. If you switch between different solvers, you will likely want to start over with your testnet/simnet dag.

## Using the lean cuckoo solver

### Requirements

* [gcc compiler and toolchain](https://gcc.gnu.org/).

### Building

Run the `build_cuckoo.sh` script. The compiled binaries are saved in your `$GOBIN` path, or in `$HOME/go/bin`.

### Running

Specify the `--lean` flag when calling soterd:
```
soterd --lean
```


## Using the mean gpu solver

### Requirements

* [CUDA-capable GPU](https://developer.nvidia.com/cuda-gpus#compute)
* [gcc compiler and toolchain](https://gcc.gnu.org/)
* [CUDA developer kit](https://developer.nvidia.com/cuda-downloads)

Linux example for installing CUDA developer kit:
```
wget http://developer.download.nvidia.com/compute/cuda/10.1/Prod/local_installers/cuda_10.1.243_418.87.00_linux.run
chmod 755 cuda_10.1.243_418.87.00_linux.run
sudo ./cuda_10.1.243_418.87.00_linux.run
```

### Google Compute Engine

External GPU solver has been successfully tested with GCE instances.
```
Region: us-west1
Instance: n1-standard-4
Image: c0-common-gce-gpu-image-20191003
Image Description: Google, GPU Optimized Debian, m32 (with CUDA 10.0), A Debian 9 based image with CUDA/CuDNN/NCCL pre-installed
GPU: NVIDIA Tesla T4
GPU Count: 1
```

The gce image streamlines the NVIDIA CUDA developer kit setup, so all that's needed is golang and soterd.

### Building

Run the `build_cuckoo_gpu.sh` script. The compiled binaries are saved in your `$GOBIN` path, or in `$HOME/go/bin`.

### Running

Specify the `--gpu` flag when calling soterd:
```
soterd --gpu
```

## Using external solvers with test suites

Test suites call `cuckoo.DefaultSolver()` from `mining/cuckoo`. To switch to an external solver for all of the test suites, the current simplest way is to modify the `mining/cuckoo/cuckoo.go` file, editing the `DefaultSolver` function to use either `NewGPUCuckooSolver()` or `NewCuckooSolver(DefaultEdgeBits, DefaultSipHash)`.

