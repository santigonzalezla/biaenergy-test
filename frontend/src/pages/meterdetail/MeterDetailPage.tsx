import {useParams} from 'react-router';
import PageHeader from '@/components/shared/pageheader/PageHeader.tsx';

const MeterDetailPage = () =>
{
    const {meterId} = useParams();

    return <PageHeader title='Detalle del medidor' description={`Medidor ${meterId}`}/>;
}

export default MeterDetailPage;
