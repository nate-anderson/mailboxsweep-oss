import { useEffect, useState } from "react";
import { API } from "../../client";
import { Classifier, FolderMessageCount } from "../../client-gen";
import Navbar from "../../components/navbar";
import Container from '@mui/material/Container';
import CssBaseline from "@mui/material/CssBaseline";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { getYear } from "date-fns";
import { SelectedClassifier } from "../../lib/classifier";

import type {} from '@mui/x-date-pickers/themeAugmentation';
import SelectedClassifierList from "../../components/classifier-inputs/list";

const year = getYear(new Date());

const defaultClassifiers: Classifier[] = [
    { typeKey: "SENDER_DOMAIN" },
    { typeKey: "SENDER_USERNAME" },
    // default to the first day of three years ago
    { typeKey: "SEND_DATE", initValue: `01/01/${year - 3}`}
];

export default function Configure() {
    const [ expandedIndex, setExpandedIndex ] = useState<number | undefined>();
    const [ error, setError ] = useState<string | undefined>();
    const [ selectedClassifiers, setSelectedClassifiers ] = useState<SelectedClassifier[]>([]);
    const [ folders, setFolders ] = useState<FolderMessageCount[]>([]);

    useEffect(() => {
        API.getClassifiers()
            .then((res) => {
                let hydratedClassifiers: SelectedClassifier[];

                try {
                    hydratedClassifiers = defaultClassifiers.map(cl => {
                        const classifier = res.classifiers.find(c => c.typeKey === cl.typeKey);
                        if (!classifier) {
                            throw new Error(`classifier with typeKey ${cl.typeKey} was not found in API response`)
                        }
    
                        return {
                            ...cl,
                            name: classifier.name,
                            description: classifier.description,
                            hasInput: classifier.hasInput,
                            enabled: true,
                            inputFormat: classifier.inputFormat,
                        }
                    });
                    setSelectedClassifiers(hydratedClassifiers);
                } catch (error) {
                    setError(error as string);
                }
            })
            .catch((err) => {
                setError(err);
            });

        API.getInboxFolders()
            .then((res) => {
                setFolders(res.folders);
            })
            .catch((err) => {
                setError(err);
            })
    }, []);

    return (
        <>
            <Navbar />
            <Container component="main">
                <CssBaseline />
                <Box sx={{
                    marginTop: 6,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center'
                }}>
                    <Typography
                        align="center"
                        sx={{
                            color: "var(--color-accent)",
                            fontSize: "2em",
                            fontWeight: 100,
                        }}
                    >
                        Let's organize your inbox.
                    </Typography>
                    <Typography
                        align="center"
                        sx={{
                            color: "var(--color-black)",
                            marginY: 2,
                            maxWidth: 600,
                            marginX: 2,
                        }}    
                    >
                        We'll put all your messages into groups based on who sent them, when they were sent, and what's inside. The recommended settings are below, but we'll organize your inbox however you like.
                    </Typography>
                </Box>
                <Box
                    sx={{
                        maxWidth: 600,
                        width: "auto",
                        marginTop: 3,
                        marginX: "auto",
                    }}
                >    
                    <SelectedClassifierList
                        setSelectedClassifiers={setSelectedClassifiers}
                        classifiers={selectedClassifiers}
                        expandedIndex={expandedIndex}
                        setExpandedIndex={setExpandedIndex}
                    />
                </Box>
            </Container>
        </>
    )
}