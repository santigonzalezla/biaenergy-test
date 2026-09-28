import {createContext, useContext} from 'react';
import {AnalysisStatus} from '@/interfaces/enums.ts';
import type {Analysis} from '@/interfaces/interfaces.ts';

export const POLL_INTERVAL_MS = 1500;

export interface AnalysisContextType {
    analysis: Analysis | null;
    isActive: boolean;
    isStarting: boolean;
    isDrawerOpen: boolean;
    completedVersion: number;
    start: (meterIds?: string[]) => Promise<void>;
    openDrawer: () => void;
    closeDrawer: () => void;
}

export const AnalysisContext = createContext<AnalysisContextType | undefined>(undefined);

export const isActiveAnalysis = (analysis: Analysis | null) =>
{
    return analysis?.status === AnalysisStatus.PENDING || analysis?.status === AnalysisStatus.RUNNING;
}

export const useAnalysis = () =>
{
    const context = useContext(AnalysisContext);

    if (context === undefined) throw new Error('useAnalysis must be used within an AnalysisProvider');

    return context;
}
