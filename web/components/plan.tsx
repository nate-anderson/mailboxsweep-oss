import Typography from "@mui/material/Typography";
import currency from "currency.js";
import Button from "@mui/material/Button";
import IconArrowForward from "@mui/icons-material/ArrowForward";
import Card from "@mui/material/Card";
import { Plan } from "../client-gen";
import Stack from "@mui/material/Stack";
import Chip from "@mui/material/Chip";

interface IPlanCardProps {
    plan: Plan;
    submit: undefined | (() => void);
    confirmText: string | undefined;
}

const discountUntil = "5/30/2024";

export function Price(props: { priceInCents: number, strikeThrough: boolean, planKey: string }) {
    return (
      <Typography
        data-testid={`price-${props.planKey.toLowerCase()}-dollars`}
        color={props.strikeThrough ? "var(--color-disabled)" : "var(--color-black)"}
        fontSize={props.strikeThrough ? "1.25em" : "1.75em"}
        sx={{
          textDecoration: props.strikeThrough ? "line-through" : "",
        }}
        marginBottom={props.strikeThrough ? 0.25 : 0}
      >
        {(props.priceInCents / 100).toLocaleString("en-US", {style:"currency", currency:"USD"})}
      </Typography>
    );
  };


export function PriceWithDiscount(props: { plan: Plan, marginTop: number }) {
  if (!props.plan.discountInCents) {
    return <Price planKey={props.plan.key} priceInCents={props.plan.priceInCents} strikeThrough={false} />;
  }

  return <>
    <Stack
      direction="row"
      columnGap={1}
      alignItems="flex-end"
      marginTop={props.marginTop}
    >
      <Price
        planKey={props.plan.key}
        strikeThrough={true}
        priceInCents={props.plan.priceInCents}
      />
      <Price
        priceInCents={props.plan.priceInCents - props.plan.discountInCents}
        planKey={props.plan.key + "_DISCOUNT"}
        strikeThrough={false}
      />
    </Stack>
    <Chip
      label={`${Math.floor((props.plan.discountInCents / props.plan.priceInCents) * 100)}% off until ${discountUntil}`}
      size="small"
      sx={{
        fontWeight: "bold",
        backgroundColor: "var(--color-highlight)",
        color: "white",
        marginY: 1,
      }}
    />
  </>
}

export default function PlanCard(props: IPlanCardProps) {
  return (
    <Card
      key={props.plan.key}
      sx={{
        maxWidth: 500,
        width: "auto",
        marginX: "auto",
        marginBottom: 2,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        padding: 3,
      }}
    >
      <Typography
        data-testid={`plan-name-${props.plan.key.toLowerCase()}`}
        variant="h5"
        align="center"
        sx={{
          color: "var(--color-accent)",
        }}
      >
        {props.plan.displayName}
      </Typography>
      <PriceWithDiscount plan={props.plan} marginTop={1}/>
      <Typography
        sx={{
          color: "var(--color-gray)",
          marginY: 0,
        }}
      >
        Pay <b>once</b> and use for {props.plan.durationDays} days
      </Typography>
      <ul
        style={{
          maxWidth: "400px",
          color: "var(--color-gray)",
        }}
      >
        {props.plan.features.map((feature, i) => {
          return <li key={i}>{feature}</li>;
        })}
      </ul>
      {
        Boolean(props.submit) ? <>
        <Button
          data-testid={`plan-${props.plan.key.toLowerCase()}-checkout`}
          size="large"
          sx={{
            maxWidth: "200px",
            marginTop: 1,
            marginBottom: 2,
            textTransform: "none",
          }}
          variant="contained"
          onClick={props.submit}
          endIcon={<IconArrowForward />}
        >
          {props.confirmText ?? "Confirm & Pay"}
        </Button>
        </> : undefined
      }
      
      
    </Card>
  );
}