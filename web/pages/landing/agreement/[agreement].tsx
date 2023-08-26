import { CssBaseline } from "@mui/material";
import Container from "@mui/material/Container";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import IconGrading from "@mui/icons-material/Grading";
import Paper from "@mui/material/Paper";
import { parseISO, format, formatISO } from "date-fns";
import ReactMarkdown from 'react-markdown'
import PageHeader from "../../../components/page-header";
import { readdir, readFile } from "fs/promises";
import TopBar from "../../../components/landing/top-bar";
import IconQuestions from "@mui/icons-material/Quiz";
import IconMarkEmailUnread from "@mui/icons-material/MarkEmailUnread";
import { displayFormat } from "../../../dates";
import { GetStaticPaths, GetStaticProps } from "next";
import { join } from "path";
import { parse as parseYaml } from "yaml";
import remarkGfm from "remark-gfm";

export const getStaticPaths: GetStaticPaths = async () => {
    const paths = await readdir(
        join(process.cwd(), "agreements"),
    );

    return {
        paths: paths.map((p) => ({ params: { agreement: p.replace(".yml", "") }})),
        fallback: false,
    }
}


export const getStaticProps = async ({ params }: { params: { agreement: string } }) => {
    const filePath = join(process.cwd(), "agreements", `${params.agreement}.yml`)
    const yamlData = await readFile(filePath, { encoding: "utf-8" });
    const agreement = parseYaml(yamlData)

    console.log(Object.keys(agreement));

    return {
        props: { agreement },
    };
}

export default function Agreement(props: any) {
    const drawerItems = [
        {
            text: "Home",
            icon: <IconMarkEmailUnread />,
            action: {
              path: "/",
            }
        },
        {
            text: "Common Questions",
            icon: <IconQuestions />,
            action: {
              path: "/landing/faq"
            },
        }
      ];

  return (
      <>
        <TopBar shadow={false} drawerItems={drawerItems} backgroundColor="#FFFFFF"/>
        <Container component="main">
          <CssBaseline />
          <Box
            sx={{
                marginY: 3,
                maxWidth: 600,
                width: "auto",
                marginX: "auto",
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
            }}
        >
        <Typography
            align="center"
            sx={{
                color: "var(--color-black)",
                fontSize: "2em",
                fontWeight: 100,
                marginBottom: 2,
            }}
        >
        {props.agreement?.name}
        </Typography>
        <Typography>{props.agreement?.description}</Typography>
    </Box>
          <Box
            sx={{
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
            }}
          >
            <Paper
              variant="outlined"
              sx={{
                margin: 1,
                paddingX: 3,
                paddingY: 1,
                fontFamily: "monospace",
                fontSize: "12px",
                lineHeight: "1.75em",
              }}>
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {props.agreement?.body ?? ""}
              </ReactMarkdown>
            </Paper>
            <Typography
              align="center"
              sx={{
                color: "var(--color-gray)",
                marginTop: 1,
                marginBottom: 3,
              }}
            >
              Terms Updated {props.agreement?.agreementUpdatedAt}
            </Typography>
          </Box>
        </Container>
      </>
  );
}
