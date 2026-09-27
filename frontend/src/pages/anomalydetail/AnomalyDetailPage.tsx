import {useParams} from 'react-router';
import PageHeader from '@/components/shared/pageheader/PageHeader.tsx';

const AnomalyDetailPage = () =>
{
    const {anomalyId} = useParams();

    return <PageHeader title='Investigación de anomalía' description={`Anomalía ${anomalyId}`}/>;
}

export default AnomalyDetailPage;
