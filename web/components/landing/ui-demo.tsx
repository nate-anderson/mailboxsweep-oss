import Image, { StaticImageData } from "next/image";
import { useEffect, useState } from "react";
import Demo1 from "../../public/ui/level_1.png";
import Demo2 from "../../public/ui/level_2.png";
import Demo3 from "../../public/ui/level_3.png";
import UIPreviewStyles from "../../styles/ui-preview.module.css";

export default function UIDemo() {
    const images: StaticImageData[] = [Demo1, Demo2, Demo3];
    const [imgIndex, setImgIndex] = useState(0);
    
    useEffect(() => {
        setInterval(() => {
            setImgIndex(ind => ind + 1);
        }, 5000);
    }, []);

    return <div className={UIPreviewStyles.device}>
        <Image
            src={images[imgIndex % 3]}
            alt="Select messages by website, sender and date"
            priority
            style={{
                width: "50vw",
                height: "auto",
                maxWidth: "270px",
            }}
        />
    </div>;
}