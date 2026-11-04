import {getConfigStatus} from 'client/client';
import React, {useEffect, useState} from 'react';

const CallbackURL = () => {
    const [callbackURL, setCallbackURL] = useState('');
    const [configured, setConfigured] = useState<boolean | null>(null);
    const [copied, setCopied] = useState(false);

    useEffect(() => {
        let active = true;
        getConfigStatus().then((status) => {
            if (active) {
                setCallbackURL(status.callback_url || '');
                setConfigured(status.configured);
            }
        }).catch(() => {
            if (active) {
                setConfigured(false);
            }
        });
        return () => {
            active = false;
        };
    }, []);

    const copyURL = async () => {
        try {
            await navigator.clipboard.writeText(callbackURL);
            setCopied(true);
        } catch {
            setCopied(false);
        }
    };

    return (
        <div>
            <p>{'No Google Cloud, abra o cliente OAuth do tipo Aplicativo da Web e adicione esta URL em “URIs de redirecionamento autorizados”:'}</p>
            {callbackURL ? (
                <>
                    <input
                        aria-label='URL de retorno OAuth para copiar'
                        readOnly={true}
                        style={{maxWidth: '100%', width: '100%'}}
                        type='text'
                        value={callbackURL}
                        onFocus={(event) => event.currentTarget.select()}
                    />
                    <button
                        disabled={!callbackURL}
                        type='button'
                        onClick={copyURL}
                    >
                        {copied ? 'Copiado' : 'Copiar URL'}
                    </button>
                </>
            ) : <p>{'Configure a URL pública do Mattermost para gerar o retorno OAuth.'}</p>}
            {configured === true && <p>{'Conexão do plugin: configurada. Os campos de credenciais podem aparecer vazios no console; não é necessário preencher novamente.'}</p>}
            {configured === false && <p>{'Conexão do plugin: configuração pendente ou status indisponível.'}</p>}
        </div>
    );
};

export default CallbackURL;
