import "@testing-library/jest-dom/vitest";

function createStorage(): Storage {
  const items = new Map<string, string>();

  return {
    get length() {
      return items.size;
    },
    clear: () => items.clear(),
    getItem: (key) => items.get(key) ?? null,
    key: (index) => Array.from(items.keys())[index] ?? null,
    removeItem: (key) => items.delete(key),
    setItem: (key, value) => items.set(key, value)
  };
}

Object.defineProperty(globalThis, "localStorage", {
  value: createStorage(),
  configurable: true
});

Object.defineProperty(globalThis, "sessionStorage", {
  value: window.sessionStorage ?? createStorage(),
  configurable: true
});
