import { useRouter } from "next/router";
import { useEffect } from "react";
import { API } from "../client";
import Container from '@mui/material/Container';


export default function Login() {
    const router = useRouter();

    useEffect(() => {
        if(!router.isReady) return;
        const query = router.query;

        const loginToken = query["token"] as string;
        const signature = query["sig"];

        API.submitLoginToken(loginToken)
            .then((res) => {
                console.log({ loginSuccess: res });
                router.push("/home");
            })
            .catch(console.error);

    }, [router.isReady, router.query]);

    return (
        <main>
            <Container component="main">
                
            </Container>
        </main>
    );
}