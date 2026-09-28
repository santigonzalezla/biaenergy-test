import styles from './meters.module.css';
import {useEffect, useState} from 'react';
import {useSearchParams} from 'react-router';
import {toast} from 'sonner';
import {Gauge, Plus, SearchX} from 'lucide-react';
import MeterFormModal from '@/components/meters/meterformmodal/MeterFormModal.tsx';
import MetersTable from '@/components/meters/meterstable/MetersTable.tsx';
import MetersToolbar from '@/components/meters/meterstoolbar/MetersToolbar.tsx';
import Button from '@/components/shared/button/Button.tsx';
import ConfirmDialog from '@/components/shared/confirmdialog/ConfirmDialog.tsx';
import PageHeader from '@/components/shared/pageheader/PageHeader.tsx';
import Pagination from '@/components/shared/pagination/Pagination.tsx';
import Skeleton from '@/components/shared/skeleton/Skeleton.tsx';
import StateMessage from '@/components/shared/statemessage/StateMessage.tsx';
import {useAnalysis} from '@/context/analysis.ts';
import {useDebouncedValue} from '@/hooks/useDebouncedValue.ts';
import {useFetch} from '@/hooks/useFetch.ts';
import {MeterSortField, MeterStatus, SortDirection} from '@/interfaces/enums.ts';
import type {Meter, Page} from '@/interfaces/interfaces.ts';
import {ApiError, apiRequest, buildQuery} from '@/lib/api.ts';

const PAGE_SIZE = 20;

type FormState = {mode: 'create'} | {mode: 'edit'; meter: Meter};

const parseEnum = <T extends string>(value: string | null, allowed: Record<string, T>): T | null =>
{
    return value && (Object.values(allowed) as string[]).includes(value) ? value as T : null;
}

const MetersPage = () =>
{
    const [params, setParams] = useSearchParams();
    const {completedVersion} = useAnalysis();

    const status = parseEnum(params.get('status'), MeterStatus);
    const sortBy = parseEnum(params.get('sortBy'), MeterSortField) ?? MeterSortField.CODE;
    const sortDir = parseEnum(params.get('sortDir'), SortDirection) ?? SortDirection.ASC;
    const page = Math.max(1, Number(params.get('page')) || 1);
    const search = params.get('search') ?? '';

    const [searchInput, setSearchInput] = useState(search);
    const debouncedSearch = useDebouncedValue(searchInput.trim());

    const [formState, setFormState] = useState<FormState | null>(null);
    const [toDelete, setToDelete] = useState<Meter | null>(null);
    const [isDeleting, setIsDeleting] = useState(false);

    const path = `/meters${buildQuery({search, status, sortBy, sortDir, page, limit: PAGE_SIZE})}`;
    const meters = useFetch<Page<Meter>>(path);
    const {execute: refresh} = meters;

    const updateParams = (changes: Record<string, string | null>, resetPage = true) =>
    {
        setParams(previous =>
        {
            const next = new URLSearchParams(previous);

            Object.entries(changes).forEach(([key, value]) => value ? next.set(key, value) : next.delete(key));

            if (resetPage) next.delete('page');

            return next;
        }, {replace: true});
    }

    useEffect(() =>
    {
        if (debouncedSearch === search) return;

        setParams(previous =>
        {
            const next = new URLSearchParams(previous);

            if (debouncedSearch) next.set('search', debouncedSearch);
            else next.delete('search');

            next.delete('page');

            return next;
        }, {replace: true});
    }, [debouncedSearch, search, setParams]);

    useEffect(() =>
    {
        if (completedVersion > 0) void refresh();
    }, [completedVersion, refresh]);

    const handleSort = (field: MeterSortField) =>
    {
        const nextDir = sortBy === field && sortDir === SortDirection.ASC ? SortDirection.DESC : SortDirection.ASC;

        updateParams({sortBy: field, sortDir: nextDir});
    }

    const handleSaved = (saved: Meter) =>
    {
        toast.success(formState?.mode === 'edit' ? `Medidor ${saved.code} actualizado` : `Medidor ${saved.code} creado`);
        setFormState(null);
        void refresh();
    }

    const confirmDelete = async () =>
    {
        if (!toDelete) return;

        setIsDeleting(true);

        try
        {
            await apiRequest<void>(`/meters/${toDelete.id}`, {method: 'DELETE'});
            toast.success(`Medidor ${toDelete.code} eliminado`);
            setToDelete(null);
            void refresh();
        }
        catch (error)
        {
            toast.error('No se pudo eliminar el medidor', {description: ApiError.from(error).message});
        }
        finally
        {
            setIsDeleting(false);
        }
    }

    const hasFilters = Boolean(search || status);
    const data = meters.data;

    return (
        <div className={styles.page}>
            <PageHeader
                title='Medidores'
                description='Puntos de medición con su estado según el último análisis y su consumo de las últimas 48 horas.'
                actions={<Button variant='primary' icon={<Plus size={16}/>} onClick={() => setFormState({mode: 'create'})}>Nuevo medidor</Button>}
            />

            <section className={styles.panel}>
                <MetersToolbar
                    search={searchInput}
                    status={status}
                    onSearchChange={setSearchInput}
                    onStatusChange={next => updateParams({status: next})}
                />

                {meters.error && !data && (
                    <StateMessage variant='error' title='No se pudieron cargar los medidores' description={meters.error.message} onRetry={() => void refresh()}/>
                )}

                {!data && !meters.error && (
                    <div className={styles.loading}>
                        {Array.from({length: 6}, (_, index) => <Skeleton key={index} height='44px'/>)}
                    </div>
                )}

                {data && data.data.length === 0 && (
                    hasFilters
                        ? <StateMessage icon={SearchX} title='Sin resultados' description='Ningún medidor coincide con la búsqueda o el filtro.'/>
                        : <StateMessage icon={Gauge} title='Aún no hay medidores' description='Crea un medidor o importa lecturas para empezar.'/>
                )}

                {data && data.data.length > 0 && (
                    <>
                        <MetersTable
                            meters={data.data}
                            sortBy={sortBy}
                            sortDir={sortDir}
                            onSort={handleSort}
                            onEdit={meter => setFormState({mode: 'edit', meter})}
                            onDelete={setToDelete}
                        />
                        <Pagination
                            page={data.page}
                            limit={data.limit}
                            total={data.total}
                            itemLabel='medidores'
                            onPageChange={next => updateParams({page: String(next)}, false)}
                        />
                    </>
                )}
            </section>

            {formState && (
                <MeterFormModal
                    key={formState.mode === 'edit' ? formState.meter.id : 'new'}
                    isOpen
                    meter={formState.mode === 'edit' ? formState.meter : null}
                    onClose={() => setFormState(null)}
                    onSaved={handleSaved}
                />
            )}

            <ConfirmDialog
                isOpen={toDelete !== null}
                title={`Eliminar ${toDelete?.code ?? ''}`}
                message='El medidor dejará de aparecer en la aplicación y no se incluirá en los próximos análisis. Sus lecturas históricas se conservan.'
                confirmLabel='Eliminar medidor'
                isLoading={isDeleting}
                onConfirm={() => void confirmDelete()}
                onCancel={() => setToDelete(null)}
            />
        </div>
    );
}

export default MetersPage;
