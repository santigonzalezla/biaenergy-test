import styles from './evidencepanel.module.css';
import {ArrowRight} from 'lucide-react';
import type {Evidence} from '@/interfaces/interfaces.ts';
import {formatNumber, formatPercent, formatShortDateTime} from '@/lib/format.ts';
import {SIGNAL_KIND, VARIABLE, variableOf} from '@/lib/labels.ts';

interface EvidencePanelProps {
    evidence: Evidence;
}

const SIGNAL_VARIABLE: Record<string, string> = {
    CONSUMPTION_SURGE: 'consumption_kwh',
    CONSUMPTION_DROP: 'consumption_kwh',
    VOLTAGE_OUT_OF_RANGE: 'voltage',
    VOLTAGE_INSTABILITY: 'voltage_step',
    LOW_POWER_FACTOR: 'power_factor',
    OVERCURRENT: 'current',
};

const withUnit = (value: number, variableName: string) =>
{
    const variable = variableOf(variableName);

    return `${formatNumber(value, variable.digits)}${variable.unit ? ` ${variable.unit}` : ''}`;
}

const EvidencePanel = ({evidence}: EvidencePanelProps) =>
{
    return (
        <div className={styles.panel}>
            <section aria-labelledby='signals-title'>
                <h3 id='signals-title' className={styles.subtitle}>Señales detectadas</h3>
                <ul className={styles.signals}>
                    {evidence.signals.map(signal =>
                    {
                        const variableName = SIGNAL_VARIABLE[signal.kind] ?? 'consumption_kwh';

                        return (
                            <li key={`${signal.kind}-${signal.startedAt}`} className={styles.signal}>
                                <div className={styles.signalHeader}>
                                    <strong>{SIGNAL_KIND[signal.kind] ?? signal.kind}</strong>
                                    <span className={styles.since}>
                                        {formatShortDateTime(signal.startedAt)}
                                        {signal.endedAt ? ` → ${formatShortDateTime(signal.endedAt)}` : ' · continúa'}
                                    </span>
                                </div>
                                <div className={`${styles.values} tabular`}>
                                    <span>{withUnit(signal.baselineValue, variableName)}</span>
                                    <ArrowRight size={14}/>
                                    <strong>{withUnit(signal.observedValue, variableName)}</strong>
                                </div>
                                {signal.description && <p className={styles.description}>{signal.description}</p>}
                            </li>
                        );
                    })}
                </ul>
            </section>

            {evidence.changedVariables.length > 0 && (
                <section aria-labelledby='variables-title'>
                    <h3 id='variables-title' className={styles.subtitle}>Variables que cambiaron</h3>
                    <table className={styles.table}>
                        <thead>
                            <tr>
                                <th scope='col'>Variable</th>
                                <th scope='col'>Antes</th>
                                <th scope='col'>Después</th>
                                <th scope='col'>Cambio</th>
                            </tr>
                        </thead>
                        <tbody>
                            {evidence.changedVariables.map(variable => (
                                <tr key={variable.name}>
                                    <th scope='row'>{VARIABLE[variable.name]?.label ?? variable.name}</th>
                                    <td className='tabular'>{withUnit(variable.baseline, variable.name)}</td>
                                    <td className='tabular'>{withUnit(variable.observed, variable.name)}</td>
                                    <td className={`tabular ${Math.abs(variable.changePct) >= 20 ? styles.strong : ''}`}>
                                        {formatPercent(variable.changePct)}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </section>
            )}
        </div>
    );
}

export default EvidencePanel;
