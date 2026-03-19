NVCC = /opt/cuda/bin/nvcc
NVCC_FLAGS = -ptx -arch=sm_120 -O3

# RTX 5090 uses compute capability 12.0 (Blackwell architecture)

KERNELS = test_kernel.ptx btc_recover.ptx

all: $(KERNELS)

%.ptx: %.cu
	$(NVCC) $(NVCC_FLAGS) $< -o $@

btc_recover.ptx: btc_recover.cu GPUMath.h GPUHash.h

clean:
	rm -f *.ptx

.PHONY: all clean
