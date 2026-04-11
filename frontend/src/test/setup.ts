import "@testing-library/jest-dom";

// jsdom may fail to initialise localStorage when --localstorage-file is not resolvable.
// Provide a reliable in-memory shim so any component that touches localStorage works in tests.
const localStorageMap = new Map<string, string>();
const localStorageMock: Storage = {
  getItem: (key) => localStorageMap.get(key) ?? null,
  setItem: (key, value) => { localStorageMap.set(key, value); },
  removeItem: (key) => { localStorageMap.delete(key); },
  clear: () => { localStorageMap.clear(); },
  key: (index) => [...localStorageMap.keys()][index] ?? null,
  get length() { return localStorageMap.size; },
};
Object.defineProperty(window, "localStorage", { value: localStorageMock, writable: true });

// ResizeObserver is not implemented in jsdom but required by Radix UI (Checkbox, etc.)
window.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};
