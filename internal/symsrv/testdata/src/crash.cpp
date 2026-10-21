// A stand-in for a native game module: a C++ method a crash lands in, and an exported entry point that calls it.
struct GfxDeviceD3D11 {
  int DrawBuffers(int *p);
};

int GfxDeviceD3D11::DrawBuffers(int *p) { return p[3] + p[7] * 2 + 1; }

extern "C" __declspec(dllexport) int UnityMain(int *p) {
  GfxDeviceD3D11 d;
  return d.DrawBuffers(p) * 3;
}
