import Typography from "@mui/material/Typography";
import currency from "currency.js";
import Button from "@mui/material/Button";
import IconArrowForward from "@mui/icons-material/ArrowForward";
import Card from "@mui/material/Card";
import { Plan } from "../client-gen";

interface IPlanCardProps {
    plan: Plan;
    submit: undefined | (() => void)
}

const renderPrice = (price: currency) => {
    const cents = String(price.cents()).padStart(2, "0");
    const commonStyles = {
      marginBottom: 0,
    };
    return (
      <div>
        <Typography
          display="inline"
          sx={{
            ...commonStyles,
            marginRight: 0,
            display: "inline-block",
            fontSize: "1.5em",
            marginY: 1,
            fontWeight: "light",
          }}
        >
          $
        </Typography>
        <Typography
          display="inline"
          sx={{
            ...commonStyles,
            fontSize: "2.5em",
            marginY: 1,
            fontWeight: "light",
          }}
        >
          {price.dollars()}
        </Typography>
        <Typography
          display="inline"
          sx={{
            ...commonStyles,
            fontSize: "2em",
            marginY: 1,
            marginLeft: -0.4,
            fontWeight: "light",
          }}
        >
          .
        </Typography>
        <Typography
          display="inline"
          sx={{
            ...commonStyles,
            display: "inline-block",
            fontSize: "1.5em",
            marginY: 1,
            fontWeight: "light",
          }}
        >
          {cents}
        </Typography>
      </div>
    );
  };

export default function PlanCard(props: IPlanCardProps) {
    const price = currency(props.plan.priceInCents, { fromCents: true });
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
        variant="h5"
        sx={{
          color: "var(--color-accent)",
        }}
      >
        {props.plan.displayName}
      </Typography>
      {renderPrice(price)}
      <Typography
        sx={{
          color: "var(--color-gray)",
          marginY: 0,
        }}
      >
        Pay once and use for {props.plan.durationDays} days
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
          Confirm & Pay
        </Button>
        <small
        style={{
          color: "var(--color-gray)",
        }}
      >
        Stripe checkout will open in a new tab.
      </small>
        </> : undefined
      }
      
      
    </Card>
  );
}