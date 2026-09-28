import {useCallback, useEffect, useMemo, useState, type ReactNode} from 'react';
import {toast} from 'sonner';
import {AnalysisContext, isActiveAnalysis, POLL_INTERVAL_MS, type AnalysisContextType} from '@/context/analysis.ts';
import {AnalysisStatus} from '@/interfaces/enums.ts';
import type {Analysis, StartAnalysisRequest} from '@/interfaces/interfaces.ts';
import {ApiError, apiRequest} from '@/lib/api.ts';

interface AnalysisProviderProps {
    children: ReactNode;
}

const AnalysisProvider = ({children}: AnalysisProviderProps) =>
{
    const [analysis, setAnalysis] = useState<Analysis | null>(null);
    const [isStarting, setIsStarting] = useState(false);
    const [isDrawerOpen, setIsDrawerOpen] = useState(false);
    const [completedVersion, setCompletedVersion] = useState(0);

    const isActive = isActiveAnalysis(analysis);

    useEffect(() =>
    {
        const controller = new AbortController();

        apiRequest<Analysis>('/ai/analysis/latest', {signal: controller.signal})
            .then(setAnalysis)
            .catch(() => undefined);

        return () => controller.abort();
    }, []);

    useEffect(() =>
    {
        if (!analysis || !isActive) return;

        const controller = new AbortController();

        const timer = setTimeout(async () =>
        {
            try
            {
                const next = await apiRequest<Analysis>(`/ai/analysis/${analysis.id}`, {signal: controller.signal});

                if (next.status === AnalysisStatus.COMPLETED)
                {
                    toast.success('Análisis completado', {description: `${next.anomalies} anomalías · ${next.highPriority} de alta prioridad`});
                    setCompletedVersion(version => version + 1);
                }

                if (next.status === AnalysisStatus.FAILED)
                {
                    toast.error('El análisis falló', {description: next.error ?? 'El servicio de IA no respondió.'});
                }

                setAnalysis(next);
            }
            catch (error)
            {
                if (controller.signal.aborted || ApiError.from(error).isSessionError) return;

                setAnalysis(previous => previous && {...previous});
            }
        }, POLL_INTERVAL_MS);

        return () =>
        {
            clearTimeout(timer);
            controller.abort();
        };
    }, [analysis, isActive]);

    const start = useCallback(async (meterIds?: string[]) =>
    {
        setIsStarting(true);
        setIsDrawerOpen(true);

        try
        {
            const body: StartAnalysisRequest | undefined = meterIds?.length ? {meterIds} : undefined;
            const started = await apiRequest<Analysis>('/ai/analyze', {method: 'POST', body});

            setAnalysis(started);
        }
        catch (error)
        {
            toast.error('No se pudo iniciar el análisis', {description: ApiError.from(error).message});
        }
        finally
        {
            setIsStarting(false);
        }
    }, []);

    const openDrawer = useCallback(() => setIsDrawerOpen(true), []);
    const closeDrawer = useCallback(() => setIsDrawerOpen(false), []);

    const value = useMemo<AnalysisContextType>(() => ({
        analysis,
        isActive,
        isStarting,
        isDrawerOpen,
        completedVersion,
        start,
        openDrawer,
        closeDrawer,
    }), [analysis, isActive, isStarting, isDrawerOpen, completedVersion, start, openDrawer, closeDrawer]);

    return (
        <AnalysisContext.Provider value={value}>
            {children}
        </AnalysisContext.Provider>
    );
}

export default AnalysisProvider;
