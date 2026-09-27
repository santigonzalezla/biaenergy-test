import {act, renderHook} from '@testing-library/react';
import type {ReactNode} from 'react';
import ThemeProvider from '@/context/ThemeContext.tsx';
import {THEME_STORAGE_KEY, useTheme} from '@/context/theme.ts';

const wrapper = ({children}: {children: ReactNode}) => <ThemeProvider>{children}</ThemeProvider>;

describe('ThemeProvider', () =>
{
    beforeEach(() =>
    {
        localStorage.clear();
        document.documentElement.removeAttribute('data-theme');
    });

    it('starts in light mode by default', () =>
    {
        const {result} = renderHook(() => useTheme(), {wrapper});

        expect(result.current.theme).toBe('light');
        expect(document.documentElement.getAttribute('data-theme')).toBe('light');
    });

    it('keeps the theme applied by the index.html script before React mounts', () =>
    {
        document.documentElement.setAttribute('data-theme', 'dark');

        const {result} = renderHook(() => useTheme(), {wrapper});

        expect(result.current.theme).toBe('dark');
    });

    it('toggles the theme, updates the document and remembers it', () =>
    {
        const {result} = renderHook(() => useTheme(), {wrapper});

        act(() => result.current.toggleTheme());

        expect(result.current.theme).toBe('dark');
        expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
        expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
    });
});
