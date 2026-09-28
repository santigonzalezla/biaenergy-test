import '@testing-library/jest-dom/vitest';
import {configure} from '@testing-library/react';

configure({asyncUtilTimeout: 5000});

class ResizeObserverStub
{
    observe() {}
    unobserve() {}
    disconnect() {}
}

globalThis.ResizeObserver ??= ResizeObserverStub as unknown as typeof ResizeObserver;

beforeEach(() =>
{
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) =>
    {
        throw new Error(`Unexpected network call in a test: ${String(input)}`);
    }));
});

afterEach(() =>
{
    vi.unstubAllGlobals();
});
