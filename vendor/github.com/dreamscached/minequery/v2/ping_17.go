package minequery

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"strings"

	"github.com/google/uuid"
)

var (
	ping17HandshakePacketID uint32 = 0
	ping17NextStateStatus   uint32 = 1

	ping17StatusRequestPacketID  uint32 = 0
	ping17StatusResponsePacketID uint32 = 0
	ping17StatusImagePrefix             = "data:image/png;base64,"
)

// Ping17ProtocolVersionUndefined holds a special value (=-1) sent in ping packet that indicates that client
// has not yet determined a version it will use to connect to server and does a preflight ping to determine it.
const Ping17ProtocolVersionUndefined int32 = -1

//goland:noinspection GoUnusedConst
const (
	// Ping17ProtocolVersion1191pre1 holds a protocol version (=1073741917) for Minecraft 1.19.1-pre1.
	Ping17ProtocolVersion1191pre1 int32 = 1073741917

	// Ping17ProtocolVersion22w24a holds a protocol version (=1073741916) for Minecraft 22w24a.
	Ping17ProtocolVersion22w24a int32 = 1073741916

	// Ping17ProtocolVersion119 holds a protocol version (=759) for Minecraft 1.19.
	Ping17ProtocolVersion119 int32 = 759

	// Ping17ProtocolVersion119rc2 holds a protocol version (=1073741915) for Minecraft 1.19-rc2.
	Ping17ProtocolVersion119rc2 int32 = 1073741915

	// Ping17ProtocolVersion119rc1 holds a protocol version (=1073741914) for Minecraft 1.19-rc1.
	Ping17ProtocolVersion119rc1 int32 = 1073741914

	// Ping17ProtocolVersion119pre5 holds a protocol version (=1073741913) for Minecraft 1.19-pre5.
	Ping17ProtocolVersion119pre5 int32 = 1073741913

	// Ping17ProtocolVersion119pre4 holds a protocol version (=1073741912) for Minecraft 1.19-pre4.
	Ping17ProtocolVersion119pre4 int32 = 1073741912

	// Ping17ProtocolVersion119pre3 holds a protocol version (=1073741911) for Minecraft 1.19-pre3.
	Ping17ProtocolVersion119pre3 int32 = 1073741911

	// Ping17ProtocolVersion119pre2 holds a protocol version (=1073741910) for Minecraft 1.19-pre2.
	Ping17ProtocolVersion119pre2 int32 = 1073741910

	// Ping17ProtocolVersion119pre1 holds a protocol version (=1073741909) for Minecraft 1.19-pre1.
	Ping17ProtocolVersion119pre1 int32 = 1073741909

	// Ping17ProtocolVersion22w19a holds a protocol version (=1073741908) for Minecraft 22w19a.
	Ping17ProtocolVersion22w19a int32 = 1073741908

	// Ping17ProtocolVersion22w18a holds a protocol version (=1073741907) for Minecraft 22w18a.
	Ping17ProtocolVersion22w18a int32 = 1073741907

	// Ping17ProtocolVersion22w17a holds a protocol version (=1073741906) for Minecraft 22w17a.
	Ping17ProtocolVersion22w17a int32 = 1073741906

	// Ping17ProtocolVersion22w16b holds a protocol version (=1073741905) for Minecraft 22w16b.
	Ping17ProtocolVersion22w16b int32 = 1073741905

	// Ping17ProtocolVersion22w16a holds a protocol version (=1073741904) for Minecraft 22w16a.
	Ping17ProtocolVersion22w16a int32 = 1073741904

	// Ping17ProtocolVersion22w15a holds a protocol version (=1073741903) for Minecraft 22w15a.
	Ping17ProtocolVersion22w15a int32 = 1073741903

	// Ping17ProtocolVersion22w14a holds a protocol version (=1073741902) for Minecraft 22w14a.
	Ping17ProtocolVersion22w14a int32 = 1073741902

	// Ping17ProtocolVersion22w13OneBlockAtATime holds a protocol version (=1073741901) for Minecraft 22w13oneBlockAtATime.
	Ping17ProtocolVersion22w13OneBlockAtATime int32 = 1073741901

	// Ping17ProtocolVersion22w13a holds a protocol version (=1073741900) for Minecraft 22w13a.
	Ping17ProtocolVersion22w13a int32 = 1073741900

	// Ping17ProtocolVersion22w12a holds a protocol version (=1073741899) for Minecraft 22w12a.
	Ping17ProtocolVersion22w12a int32 = 1073741899

	// Ping17ProtocolVersion22w11a holds a protocol version (=1073741898) for Minecraft 22w11a.
	Ping17ProtocolVersion22w11a int32 = 1073741898

	// Ping17ProtocolVersion1182 holds a protocol version (=758) for Minecraft 1.18.2.
	Ping17ProtocolVersion1182 int32 = 758

	// Ping17ProtocolVersion1182pre1 holds a protocol version (=1073741894) for Minecraft 1.18.2-pre1.
	Ping17ProtocolVersion1182pre1 int32 = 1073741894

	// Ping17ProtocolVersion119exp1 holds a protocol version (=1073741893) for Minecraft 1.19-exp1.
	Ping17ProtocolVersion119exp1 int32 = 1073741893

	// Ping17ProtocolVersion22w07a holds a protocol version (=1073741892) for Minecraft 22w07a.
	Ping17ProtocolVersion22w07a int32 = 1073741892

	// Ping17ProtocolVersion22w06a holds a protocol version (=1073741891) for Minecraft 22w06a.
	Ping17ProtocolVersion22w06a int32 = 1073741891

	// Ping17ProtocolVersion22w05a holds a protocol version (=1073741890) for Minecraft 22w05a.
	Ping17ProtocolVersion22w05a int32 = 1073741890

	// Ping17ProtocolVersion22w03a holds a protocol version (=1073741889) for Minecraft 22w03a.
	Ping17ProtocolVersion22w03a int32 = 1073741889

	// Ping17ProtocolVersion1181 holds a protocol version (=757) for Minecraft 1.18.1.
	Ping17ProtocolVersion1181 int32 = 757

	// Ping17ProtocolVersion1181rc3 holds a protocol version (=1073741888) for Minecraft 1.18.1-rc3.
	Ping17ProtocolVersion1181rc3 int32 = 1073741888

	// Ping17ProtocolVersion1181rc2 holds a protocol version (=1073741887) for Minecraft 1.18.1-rc2.
	Ping17ProtocolVersion1181rc2 int32 = 1073741887

	// Ping17ProtocolVersion1181rc1 holds a protocol version (=1073741886) for Minecraft 1.18.1-rc1.
	Ping17ProtocolVersion1181rc1 int32 = 1073741886

	// Ping17ProtocolVersion1181pre1 holds a protocol version (=1073741885) for Minecraft 1.18.1-pre1.
	Ping17ProtocolVersion1181pre1 int32 = 1073741885

	// Ping17ProtocolVersion118 holds a protocol version (=757) for Minecraft 1.18.
	Ping17ProtocolVersion118 int32 = 757

	// Ping17ProtocolVersion118rc4 holds a protocol version (=1073741884) for Minecraft 1.18-rc4.
	Ping17ProtocolVersion118rc4 int32 = 1073741884

	// Ping17ProtocolVersion118rc3 holds a protocol version (=1073741883) for Minecraft 1.18-rc3.
	Ping17ProtocolVersion118rc3 int32 = 1073741883

	// Ping17ProtocolVersion118rc2 holds a protocol version (=1073741882) for Minecraft 1.18-rc2.
	Ping17ProtocolVersion118rc2 int32 = 1073741882

	// Ping17ProtocolVersion118rc1 holds a protocol version (=1073741881) for Minecraft 1.18-rc1.
	Ping17ProtocolVersion118rc1 int32 = 1073741881

	// Ping17ProtocolVersion118pre8 holds a protocol version (=1073741880) for Minecraft 1.18-pre8.
	Ping17ProtocolVersion118pre8 int32 = 1073741880

	// Ping17ProtocolVersion118pre7 holds a protocol version (=1073741879) for Minecraft 1.18-pre7.
	Ping17ProtocolVersion118pre7 int32 = 1073741879

	// Ping17ProtocolVersion118pre6 holds a protocol version (=1073741878) for Minecraft 1.18-pre6.
	Ping17ProtocolVersion118pre6 int32 = 1073741878

	// Ping17ProtocolVersion118pre5 holds a protocol version (=1073741877) for Minecraft 1.18-pre5.
	Ping17ProtocolVersion118pre5 int32 = 1073741877

	// Ping17ProtocolVersion118pre4 holds a protocol version (=1073741876) for Minecraft 1.18-pre4.
	Ping17ProtocolVersion118pre4 int32 = 1073741876

	// Ping17ProtocolVersion118pre3 holds a protocol version (=1073741875) for Minecraft 1.18-pre3.
	Ping17ProtocolVersion118pre3 int32 = 1073741875

	// Ping17ProtocolVersion118pre2 holds a protocol version (=1073741874) for Minecraft 1.18-pre2.
	Ping17ProtocolVersion118pre2 int32 = 1073741874

	// Ping17ProtocolVersion118pre1 holds a protocol version (=1073741873) for Minecraft 1.18-pre1.
	Ping17ProtocolVersion118pre1 int32 = 1073741873

	// Ping17ProtocolVersion21w44a holds a protocol version (=1073741872) for Minecraft 21w44a.
	Ping17ProtocolVersion21w44a int32 = 1073741872

	// Ping17ProtocolVersion21w43a holds a protocol version (=1073741871) for Minecraft 21w43a.
	Ping17ProtocolVersion21w43a int32 = 1073741871

	// Ping17ProtocolVersion21w42a holds a protocol version (=1073741870) for Minecraft 21w42a.
	Ping17ProtocolVersion21w42a int32 = 1073741870

	// Ping17ProtocolVersion21w41a holds a protocol version (=1073741869) for Minecraft 21w41a.
	Ping17ProtocolVersion21w41a int32 = 1073741869

	// Ping17ProtocolVersion21w40a holds a protocol version (=1073741868) for Minecraft 21w40a.
	Ping17ProtocolVersion21w40a int32 = 1073741868

	// Ping17ProtocolVersion21w39a holds a protocol version (=1073741867) for Minecraft 21w39a.
	Ping17ProtocolVersion21w39a int32 = 1073741867

	// Ping17ProtocolVersion21w38a holds a protocol version (=1073741866) for Minecraft 21w38a.
	Ping17ProtocolVersion21w38a int32 = 1073741866

	// Ping17ProtocolVersion21w37a holds a protocol version (=1073741865) for Minecraft 21w37a.
	Ping17ProtocolVersion21w37a int32 = 1073741865

	// Ping17ProtocolVersion118exp7 holds a protocol version (=1073741871) for Minecraft 1.18-exp7.
	Ping17ProtocolVersion118exp7 int32 = 1073741871

	// Ping17ProtocolVersion118exp6 holds a protocol version (=1073741870) for Minecraft 1.18-exp6.
	Ping17ProtocolVersion118exp6 int32 = 1073741870

	// Ping17ProtocolVersion118exp5 holds a protocol version (=1073741869) for Minecraft 1.18-exp5.
	Ping17ProtocolVersion118exp5 int32 = 1073741869

	// Ping17ProtocolVersion118exp4 holds a protocol version (=1073741868) for Minecraft 1.18-exp4.
	Ping17ProtocolVersion118exp4 int32 = 1073741868

	// Ping17ProtocolVersion118exp3 holds a protocol version (=1073741867) for Minecraft 1.18-exp3.
	Ping17ProtocolVersion118exp3 int32 = 1073741867

	// Ping17ProtocolVersion118exp2 holds a protocol version (=1073741866) for Minecraft 1.18-exp2.
	Ping17ProtocolVersion118exp2 int32 = 1073741866

	// Ping17ProtocolVersion118exp1 holds a protocol version (=1073741865) for Minecraft 1.18-exp1.
	Ping17ProtocolVersion118exp1 int32 = 1073741865

	// Ping17ProtocolVersion1171 holds a protocol version (=756) for Minecraft 1.17.1.
	Ping17ProtocolVersion1171 int32 = 756

	// Ping17ProtocolVersion1171rc2 holds a protocol version (=1073741864) for Minecraft 1.17.1-rc2.
	Ping17ProtocolVersion1171rc2 int32 = 1073741864

	// Ping17ProtocolVersion1171rc1 holds a protocol version (=1073741863) for Minecraft 1.17.1-rc1.
	Ping17ProtocolVersion1171rc1 int32 = 1073741863

	// Ping17ProtocolVersion1171pre3 holds a protocol version (=1073741862) for Minecraft 1.17.1-pre3.
	Ping17ProtocolVersion1171pre3 int32 = 1073741862

	// Ping17ProtocolVersion1171pre2 holds a protocol version (=1073741861) for Minecraft 1.17.1-pre2.
	Ping17ProtocolVersion1171pre2 int32 = 1073741861

	// Ping17ProtocolVersion1171pre1 holds a protocol version (=1073741860) for Minecraft 1.17.1-pre1.
	Ping17ProtocolVersion1171pre1 int32 = 1073741860

	// Ping17ProtocolVersion117 holds a protocol version (=755) for Minecraft 1.17.
	Ping17ProtocolVersion117 int32 = 755

	// Ping17ProtocolVersion117rc2 holds a protocol version (=1073741859) for Minecraft 1.17-rc2.
	Ping17ProtocolVersion117rc2 int32 = 1073741859

	// Ping17ProtocolVersion117rc1 holds a protocol version (=1073741858) for Minecraft 1.17-rc1.
	Ping17ProtocolVersion117rc1 int32 = 1073741858

	// Ping17ProtocolVersion117pre5 holds a protocol version (=1073741857) for Minecraft 1.17-pre5.
	Ping17ProtocolVersion117pre5 int32 = 1073741857

	// Ping17ProtocolVersion117pre4 holds a protocol version (=1073741856) for Minecraft 1.17-pre4.
	Ping17ProtocolVersion117pre4 int32 = 1073741856

	// Ping17ProtocolVersion117pre3 holds a protocol version (=1073741855) for Minecraft 1.17-pre3.
	Ping17ProtocolVersion117pre3 int32 = 1073741855

	// Ping17ProtocolVersion117pre2 holds a protocol version (=1073741854) for Minecraft 1.17-pre2.
	Ping17ProtocolVersion117pre2 int32 = 1073741854

	// Ping17ProtocolVersion117pre1 holds a protocol version (=1073741853) for Minecraft 1.17-pre1.
	Ping17ProtocolVersion117pre1 int32 = 1073741853

	// Ping17ProtocolVersion21w20a holds a protocol version (=1073741852) for Minecraft 21w20a.
	Ping17ProtocolVersion21w20a int32 = 1073741852

	// Ping17ProtocolVersion21w19a holds a protocol version (=1073741851) for Minecraft 21w19a.
	Ping17ProtocolVersion21w19a int32 = 1073741851

	// Ping17ProtocolVersion21w18a holds a protocol version (=1073741850) for Minecraft 21w18a.
	Ping17ProtocolVersion21w18a int32 = 1073741850

	// Ping17ProtocolVersion21w17a holds a protocol version (=1073741849) for Minecraft 21w17a.
	Ping17ProtocolVersion21w17a int32 = 1073741849

	// Ping17ProtocolVersion21w16a holds a protocol version (=1073741847) for Minecraft 21w16a.
	Ping17ProtocolVersion21w16a int32 = 1073741847

	// Ping17ProtocolVersion21w15a holds a protocol version (=1073741846) for Minecraft 21w15a.
	Ping17ProtocolVersion21w15a int32 = 1073741846

	// Ping17ProtocolVersion21w14a holds a protocol version (=1073741845) for Minecraft 21w14a.
	Ping17ProtocolVersion21w14a int32 = 1073741845

	// Ping17ProtocolVersion21w13a holds a protocol version (=1073741844) for Minecraft 21w13a.
	Ping17ProtocolVersion21w13a int32 = 1073741844

	// Ping17ProtocolVersion21w11a holds a protocol version (=1073741843) for Minecraft 21w11a.
	Ping17ProtocolVersion21w11a int32 = 1073741843

	// Ping17ProtocolVersion21w10a holds a protocol version (=1073741842) for Minecraft 21w10a.
	Ping17ProtocolVersion21w10a int32 = 1073741842

	// Ping17ProtocolVersion21w08b holds a protocol version (=1073741841) for Minecraft 21w08b.
	Ping17ProtocolVersion21w08b int32 = 1073741841

	// Ping17ProtocolVersion21w08a holds a protocol version (=1073741840) for Minecraft 21w08a.
	Ping17ProtocolVersion21w08a int32 = 1073741840

	// Ping17ProtocolVersion21w07a holds a protocol version (=1073741839) for Minecraft 21w07a.
	Ping17ProtocolVersion21w07a int32 = 1073741839

	// Ping17ProtocolVersion21w06a holds a protocol version (=1073741838) for Minecraft 21w06a.
	Ping17ProtocolVersion21w06a int32 = 1073741838

	// Ping17ProtocolVersion21w05b holds a protocol version (=1073741837) for Minecraft 21w05b.
	Ping17ProtocolVersion21w05b int32 = 1073741837

	// Ping17ProtocolVersion21w05a holds a protocol version (=1073741836) for Minecraft 21w05a.
	Ping17ProtocolVersion21w05a int32 = 1073741836

	// Ping17ProtocolVersion21w03a holds a protocol version (=1073741835) for Minecraft 21w03a.
	Ping17ProtocolVersion21w03a int32 = 1073741835

	// Ping17ProtocolVersion1165 holds a protocol version (=754) for Minecraft 1.16.5.
	Ping17ProtocolVersion1165 int32 = 754

	// Ping17ProtocolVersion1165rc1 holds a protocol version (=1073741834) for Minecraft 1.16.5-rc1.
	Ping17ProtocolVersion1165rc1 int32 = 1073741834

	// Ping17ProtocolVersion20w51a holds a protocol version (=1073741833) for Minecraft 20w51a.
	Ping17ProtocolVersion20w51a int32 = 1073741833

	// Ping17ProtocolVersion20w49a holds a protocol version (=1073741832) for Minecraft 20w49a.
	Ping17ProtocolVersion20w49a int32 = 1073741832

	// Ping17ProtocolVersion20w48a holds a protocol version (=1073741831) for Minecraft 20w48a.
	Ping17ProtocolVersion20w48a int32 = 1073741831

	// Ping17ProtocolVersion20w46a holds a protocol version (=1073741830) for Minecraft 20w46a.
	Ping17ProtocolVersion20w46a int32 = 1073741830

	// Ping17ProtocolVersion20w45a holds a protocol version (=1073741829) for Minecraft 20w45a.
	Ping17ProtocolVersion20w45a int32 = 1073741829

	// Ping17ProtocolVersion1164 holds a protocol version (=754) for Minecraft 1.16.4.
	Ping17ProtocolVersion1164 int32 = 754

	// Ping17ProtocolVersion1164rc1 holds a protocol version (=1073741827) for Minecraft 1.16.4-rc1.
	Ping17ProtocolVersion1164rc1 int32 = 1073741827

	// Ping17ProtocolVersion1164pre2 holds a protocol version (=1073741826) for Minecraft 1.16.4-pre2.
	Ping17ProtocolVersion1164pre2 int32 = 1073741826

	// Ping17ProtocolVersion1164pre1 holds a protocol version (=1073741825) for Minecraft 1.16.4-pre1.
	Ping17ProtocolVersion1164pre1 int32 = 1073741825

	// Ping17ProtocolVersion1163 holds a protocol version (=753) for Minecraft 1.16.3.
	Ping17ProtocolVersion1163 int32 = 753

	// Ping17ProtocolVersion1163rc1 holds a protocol version (=752) for Minecraft 1.16.3-rc1.
	Ping17ProtocolVersion1163rc1 int32 = 752

	// Ping17ProtocolVersion1162 holds a protocol version (=751) for Minecraft 1.16.2.
	Ping17ProtocolVersion1162 int32 = 751

	// Ping17ProtocolVersion1162rc2 holds a protocol version (=750) for Minecraft 1.16.2-rc2.
	Ping17ProtocolVersion1162rc2 int32 = 750

	// Ping17ProtocolVersion1162rc1 holds a protocol version (=749) for Minecraft 1.16.2-rc1.
	Ping17ProtocolVersion1162rc1 int32 = 749

	// Ping17ProtocolVersion1162pre3 holds a protocol version (=748) for Minecraft 1.16.2-pre3.
	Ping17ProtocolVersion1162pre3 int32 = 748

	// Ping17ProtocolVersion1162pre2 holds a protocol version (=746) for Minecraft 1.16.2-pre2.
	Ping17ProtocolVersion1162pre2 int32 = 746

	// Ping17ProtocolVersion1162pre1 holds a protocol version (=744) for Minecraft 1.16.2-pre1.
	Ping17ProtocolVersion1162pre1 int32 = 744

	// Ping17ProtocolVersion20w30a holds a protocol version (=743) for Minecraft 20w30a.
	Ping17ProtocolVersion20w30a int32 = 743

	// Ping17ProtocolVersion20w29a holds a protocol version (=741) for Minecraft 20w29a.
	Ping17ProtocolVersion20w29a int32 = 741

	// Ping17ProtocolVersion20w28a holds a protocol version (=740) for Minecraft 20w28a.
	Ping17ProtocolVersion20w28a int32 = 740

	// Ping17ProtocolVersion20w27a holds a protocol version (=738) for Minecraft 20w27a.
	Ping17ProtocolVersion20w27a int32 = 738

	// Ping17ProtocolVersion1161 holds a protocol version (=736) for Minecraft 1.16.1.
	Ping17ProtocolVersion1161 int32 = 736

	// Ping17ProtocolVersion116 holds a protocol version (=735) for Minecraft 1.16.
	Ping17ProtocolVersion116 int32 = 735

	// Ping17ProtocolVersion116rc1 holds a protocol version (=734) for Minecraft 1.16-rc1.
	Ping17ProtocolVersion116rc1 int32 = 734

	// Ping17ProtocolVersion116pre8 holds a protocol version (=733) for Minecraft 1.16-pre8.
	Ping17ProtocolVersion116pre8 int32 = 733

	// Ping17ProtocolVersion116pre7 holds a protocol version (=732) for Minecraft 1.16-pre7.
	Ping17ProtocolVersion116pre7 int32 = 732

	// Ping17ProtocolVersion116pre6 holds a protocol version (=730) for Minecraft 1.16-pre6.
	Ping17ProtocolVersion116pre6 int32 = 730

	// Ping17ProtocolVersion116pre5 holds a protocol version (=729) for Minecraft 1.16-pre5.
	Ping17ProtocolVersion116pre5 int32 = 729

	// Ping17ProtocolVersion116pre4 holds a protocol version (=727) for Minecraft 1.16-pre4.
	Ping17ProtocolVersion116pre4 int32 = 727

	// Ping17ProtocolVersion116pre3 holds a protocol version (=725) for Minecraft 1.16-pre3.
	Ping17ProtocolVersion116pre3 int32 = 725

	// Ping17ProtocolVersion116pre2 holds a protocol version (=722) for Minecraft 1.16-pre2.
	Ping17ProtocolVersion116pre2 int32 = 722

	// Ping17ProtocolVersion116pre1 holds a protocol version (=721) for Minecraft 1.16-pre1.
	Ping17ProtocolVersion116pre1 int32 = 721

	// Ping17ProtocolVersion20w22a holds a protocol version (=719) for Minecraft 20w22a.
	Ping17ProtocolVersion20w22a int32 = 719

	// Ping17ProtocolVersion20w21a holds a protocol version (=718) for Minecraft 20w21a.
	Ping17ProtocolVersion20w21a int32 = 718

	// Ping17ProtocolVersion20w20b holds a protocol version (=717) for Minecraft 20w20b.
	Ping17ProtocolVersion20w20b int32 = 717

	// Ping17ProtocolVersion20w20a holds a protocol version (=716) for Minecraft 20w20a.
	Ping17ProtocolVersion20w20a int32 = 716

	// Ping17ProtocolVersion20w19a holds a protocol version (=715) for Minecraft 20w19a.
	Ping17ProtocolVersion20w19a int32 = 715

	// Ping17ProtocolVersion20w18a holds a protocol version (=714) for Minecraft 20w18a.
	Ping17ProtocolVersion20w18a int32 = 714

	// Ping17ProtocolVersion20w17a holds a protocol version (=713) for Minecraft 20w17a.
	Ping17ProtocolVersion20w17a int32 = 713

	// Ping17ProtocolVersion20w16a holds a protocol version (=712) for Minecraft 20w16a.
	Ping17ProtocolVersion20w16a int32 = 712

	// Ping17ProtocolVersion20w15a holds a protocol version (=711) for Minecraft 20w15a.
	Ping17ProtocolVersion20w15a int32 = 711

	// Ping17ProtocolVersion20w14a holds a protocol version (=710) for Minecraft 20w14a.
	Ping17ProtocolVersion20w14a int32 = 710

	// Ping17ProtocolVersion20w14Inf holds a protocol version (=709) for Minecraft 20w14∞.
	Ping17ProtocolVersion20w14Inf int32 = 709

	// Ping17ProtocolVersion20w13b holds a protocol version (=709) for Minecraft 20w13b.
	Ping17ProtocolVersion20w13b int32 = 709

	// Ping17ProtocolVersion20w13a holds a protocol version (=708) for Minecraft 20w13a.
	Ping17ProtocolVersion20w13a int32 = 708

	// Ping17ProtocolVersion20w12a holds a protocol version (=707) for Minecraft 20w12a.
	Ping17ProtocolVersion20w12a int32 = 707

	// Ping17ProtocolVersion20w11a holds a protocol version (=706) for Minecraft 20w11a.
	Ping17ProtocolVersion20w11a int32 = 706

	// Ping17ProtocolVersion20w10a holds a protocol version (=705) for Minecraft 20w10a.
	Ping17ProtocolVersion20w10a int32 = 705

	// Ping17ProtocolVersion20w09a holds a protocol version (=704) for Minecraft 20w09a.
	Ping17ProtocolVersion20w09a int32 = 704

	// Ping17ProtocolVersion20w08a holds a protocol version (=703) for Minecraft 20w08a.
	Ping17ProtocolVersion20w08a int32 = 703

	// Ping17ProtocolVersion20w07a holds a protocol version (=702) for Minecraft 20w07a.
	Ping17ProtocolVersion20w07a int32 = 702

	// Ping17ProtocolVersion20w06a holds a protocol version (=701) for Minecraft 20w06a.
	Ping17ProtocolVersion20w06a int32 = 701

	// Ping17ProtocolVersion1152 holds a protocol version (=578) for Minecraft 1.15.2.
	Ping17ProtocolVersion1152 int32 = 578

	// Ping17ProtocolVersion1152pre2 holds a protocol version (=577) for Minecraft 1.15.2-pre2.
	Ping17ProtocolVersion1152pre2 int32 = 577

	// Ping17ProtocolVersion1152pre1 holds a protocol version (=576) for Minecraft 1.15.2-pre1.
	Ping17ProtocolVersion1152pre1 int32 = 576

	// Ping17ProtocolVersion1151 holds a protocol version (=575) for Minecraft 1.15.1.
	Ping17ProtocolVersion1151 int32 = 575

	// Ping17ProtocolVersion1151pre1 holds a protocol version (=574) for Minecraft 1.15.1-pre1.
	Ping17ProtocolVersion1151pre1 int32 = 574

	// Ping17ProtocolVersion115 holds a protocol version (=573) for Minecraft 1.15.
	Ping17ProtocolVersion115 int32 = 573

	// Ping17ProtocolVersion115pre7 holds a protocol version (=572) for Minecraft 1.15-pre7.
	Ping17ProtocolVersion115pre7 int32 = 572

	// Ping17ProtocolVersion115pre6 holds a protocol version (=571) for Minecraft 1.15-pre6.
	Ping17ProtocolVersion115pre6 int32 = 571

	// Ping17ProtocolVersion115pre5 holds a protocol version (=570) for Minecraft 1.15-pre5.
	Ping17ProtocolVersion115pre5 int32 = 570

	// Ping17ProtocolVersion115pre4 holds a protocol version (=569) for Minecraft 1.15-pre4.
	Ping17ProtocolVersion115pre4 int32 = 569

	// Ping17ProtocolVersion115pre3 holds a protocol version (=567) for Minecraft 1.15-pre3.
	Ping17ProtocolVersion115pre3 int32 = 567

	// Ping17ProtocolVersion115pre2 holds a protocol version (=566) for Minecraft 1.15-pre2.
	Ping17ProtocolVersion115pre2 int32 = 566

	// Ping17ProtocolVersion115pre1 holds a protocol version (=565) for Minecraft 1.15-pre1.
	Ping17ProtocolVersion115pre1 int32 = 565

	// Ping17ProtocolVersion19w46b holds a protocol version (=564) for Minecraft 19w46b.
	Ping17ProtocolVersion19w46b int32 = 564

	// Ping17ProtocolVersion19w46a holds a protocol version (=563) for Minecraft 19w46a.
	Ping17ProtocolVersion19w46a int32 = 563

	// Ping17ProtocolVersion19w45b holds a protocol version (=562) for Minecraft 19w45b.
	Ping17ProtocolVersion19w45b int32 = 562

	// Ping17ProtocolVersion19w45a holds a protocol version (=561) for Minecraft 19w45a.
	Ping17ProtocolVersion19w45a int32 = 561

	// Ping17ProtocolVersion19w44a holds a protocol version (=560) for Minecraft 19w44a.
	Ping17ProtocolVersion19w44a int32 = 560

	// Ping17ProtocolVersion19w42a holds a protocol version (=559) for Minecraft 19w42a.
	Ping17ProtocolVersion19w42a int32 = 559

	// Ping17ProtocolVersion19w41a holds a protocol version (=558) for Minecraft 19w41a.
	Ping17ProtocolVersion19w41a int32 = 558

	// Ping17ProtocolVersion19w40a holds a protocol version (=557) for Minecraft 19w40a.
	Ping17ProtocolVersion19w40a int32 = 557

	// Ping17ProtocolVersion19w39a holds a protocol version (=556) for Minecraft 19w39a.
	Ping17ProtocolVersion19w39a int32 = 556

	// Ping17ProtocolVersion19w38b holds a protocol version (=555) for Minecraft 19w38b.
	Ping17ProtocolVersion19w38b int32 = 555

	// Ping17ProtocolVersion19w38a holds a protocol version (=554) for Minecraft 19w38a.
	Ping17ProtocolVersion19w38a int32 = 554

	// Ping17ProtocolVersion19w37a holds a protocol version (=553) for Minecraft 19w37a.
	Ping17ProtocolVersion19w37a int32 = 553

	// Ping17ProtocolVersion19w36a holds a protocol version (=552) for Minecraft 19w36a.
	Ping17ProtocolVersion19w36a int32 = 552

	// Ping17ProtocolVersion19w35a holds a protocol version (=551) for Minecraft 19w35a.
	Ping17ProtocolVersion19w35a int32 = 551

	// Ping17ProtocolVersion19w34a holds a protocol version (=550) for Minecraft 19w34a.
	Ping17ProtocolVersion19w34a int32 = 550

	// Ping17ProtocolVersion1144 holds a protocol version (=498) for Minecraft 1.14.4.
	Ping17ProtocolVersion1144 int32 = 498

	// Ping17ProtocolVersion1144pre7 holds a protocol version (=497) for Minecraft 1.14.4-pre7.
	Ping17ProtocolVersion1144pre7 int32 = 497

	// Ping17ProtocolVersion1144pre6 holds a protocol version (=496) for Minecraft 1.14.4-pre6.
	Ping17ProtocolVersion1144pre6 int32 = 496

	// Ping17ProtocolVersion1144pre5 holds a protocol version (=495) for Minecraft 1.14.4-pre5.
	Ping17ProtocolVersion1144pre5 int32 = 495

	// Ping17ProtocolVersion1144pre4 holds a protocol version (=494) for Minecraft 1.14.4-pre4.
	Ping17ProtocolVersion1144pre4 int32 = 494

	// Ping17ProtocolVersion1144pre3 holds a protocol version (=493) for Minecraft 1.14.4-pre3.
	Ping17ProtocolVersion1144pre3 int32 = 493

	// Ping17ProtocolVersion1144pre2 holds a protocol version (=492) for Minecraft 1.14.4-pre2.
	Ping17ProtocolVersion1144pre2 int32 = 492

	// Ping17ProtocolVersion1144pre1 holds a protocol version (=491) for Minecraft 1.14.4-pre1.
	Ping17ProtocolVersion1144pre1 int32 = 491

	// Ping17ProtocolVersion1143 holds a protocol version (=490) for Minecraft 1.14.3.
	Ping17ProtocolVersion1143 int32 = 490

	// Ping17ProtocolVersion1143CombatTest holds a protocol version (=500) for Minecraft 1.14.3 - Combat Test.
	Ping17ProtocolVersion1143CombatTest int32 = 500

	// Ping17ProtocolVersion1143pre4 holds a protocol version (=489) for Minecraft 1.14.3-pre4.
	Ping17ProtocolVersion1143pre4 int32 = 489

	// Ping17ProtocolVersion1143pre3 holds a protocol version (=488) for Minecraft 1.14.3-pre3.
	Ping17ProtocolVersion1143pre3 int32 = 488

	// Ping17ProtocolVersion1143pre2 holds a protocol version (=487) for Minecraft 1.14.3-pre2.
	Ping17ProtocolVersion1143pre2 int32 = 487

	// Ping17ProtocolVersion1143pre1 holds a protocol version (=486) for Minecraft 1.14.3-pre1.
	Ping17ProtocolVersion1143pre1 int32 = 486

	// Ping17ProtocolVersion1142 holds a protocol version (=485) for Minecraft 1.14.2.
	Ping17ProtocolVersion1142 int32 = 485

	// Ping17ProtocolVersion1142pre4 holds a protocol version (=484) for Minecraft 1.14.2-pre4.
	Ping17ProtocolVersion1142pre4 int32 = 484

	// Ping17ProtocolVersion1142pre3 holds a protocol version (=483) for Minecraft 1.14.2-pre3.
	Ping17ProtocolVersion1142pre3 int32 = 483

	// Ping17ProtocolVersion1142pre2 holds a protocol version (=482) for Minecraft 1.14.2-pre2.
	Ping17ProtocolVersion1142pre2 int32 = 482

	// Ping17ProtocolVersion1142pre1 holds a protocol version (=481) for Minecraft 1.14.2-pre1.
	Ping17ProtocolVersion1142pre1 int32 = 481

	// Ping17ProtocolVersion1141 holds a protocol version (=480) for Minecraft 1.14.1.
	Ping17ProtocolVersion1141 int32 = 480

	// Ping17ProtocolVersion1141pre2 holds a protocol version (=479) for Minecraft 1.14.1-pre2.
	Ping17ProtocolVersion1141pre2 int32 = 479

	// Ping17ProtocolVersion1141pre1 holds a protocol version (=478) for Minecraft 1.14.1-pre1.
	Ping17ProtocolVersion1141pre1 int32 = 478

	// Ping17ProtocolVersion114 holds a protocol version (=477) for Minecraft 1.14.
	Ping17ProtocolVersion114 int32 = 477

	// Ping17ProtocolVersion114pre5 holds a protocol version (=476) for Minecraft 1.14-pre5.
	Ping17ProtocolVersion114pre5 int32 = 476

	// Ping17ProtocolVersion114pre4 holds a protocol version (=475) for Minecraft 1.14-pre4.
	Ping17ProtocolVersion114pre4 int32 = 475

	// Ping17ProtocolVersion114pre3 holds a protocol version (=474) for Minecraft 1.14-pre3.
	Ping17ProtocolVersion114pre3 int32 = 474

	// Ping17ProtocolVersion114pre2 holds a protocol version (=473) for Minecraft 1.14-pre2.
	Ping17ProtocolVersion114pre2 int32 = 473

	// Ping17ProtocolVersion114pre1 holds a protocol version (=472) for Minecraft 1.14-pre1.
	Ping17ProtocolVersion114pre1 int32 = 472

	// Ping17ProtocolVersion19w14b holds a protocol version (=471) for Minecraft 19w14b.
	Ping17ProtocolVersion19w14b int32 = 471

	// Ping17ProtocolVersion19w14a holds a protocol version (=470) for Minecraft 19w14a.
	Ping17ProtocolVersion19w14a int32 = 470

	// Ping17ProtocolVersion19w13b holds a protocol version (=469) for Minecraft 19w13b.
	Ping17ProtocolVersion19w13b int32 = 469

	// Ping17ProtocolVersion19w13a holds a protocol version (=468) for Minecraft 19w13a.
	Ping17ProtocolVersion19w13a int32 = 468

	// Ping17ProtocolVersion19w12b holds a protocol version (=467) for Minecraft 19w12b.
	Ping17ProtocolVersion19w12b int32 = 467

	// Ping17ProtocolVersion19w12a holds a protocol version (=466) for Minecraft 19w12a.
	Ping17ProtocolVersion19w12a int32 = 466

	// Ping17ProtocolVersion19w11b holds a protocol version (=465) for Minecraft 19w11b.
	Ping17ProtocolVersion19w11b int32 = 465

	// Ping17ProtocolVersion19w11a holds a protocol version (=464) for Minecraft 19w11a.
	Ping17ProtocolVersion19w11a int32 = 464

	// Ping17ProtocolVersion19w09a holds a protocol version (=463) for Minecraft 19w09a.
	Ping17ProtocolVersion19w09a int32 = 463

	// Ping17ProtocolVersion19w08b holds a protocol version (=462) for Minecraft 19w08b.
	Ping17ProtocolVersion19w08b int32 = 462

	// Ping17ProtocolVersion19w08a holds a protocol version (=461) for Minecraft 19w08a.
	Ping17ProtocolVersion19w08a int32 = 461

	// Ping17ProtocolVersion19w07a holds a protocol version (=460) for Minecraft 19w07a.
	Ping17ProtocolVersion19w07a int32 = 460

	// Ping17ProtocolVersion19w06a holds a protocol version (=459) for Minecraft 19w06a.
	Ping17ProtocolVersion19w06a int32 = 459

	// Ping17ProtocolVersion19w05a holds a protocol version (=458) for Minecraft 19w05a.
	Ping17ProtocolVersion19w05a int32 = 458

	// Ping17ProtocolVersion19w04b holds a protocol version (=457) for Minecraft 19w04b.
	Ping17ProtocolVersion19w04b int32 = 457

	// Ping17ProtocolVersion19w04a holds a protocol version (=456) for Minecraft 19w04a.
	Ping17ProtocolVersion19w04a int32 = 456

	// Ping17ProtocolVersion19w03c holds a protocol version (=455) for Minecraft 19w03c.
	Ping17ProtocolVersion19w03c int32 = 455

	// Ping17ProtocolVersion19w03b holds a protocol version (=454) for Minecraft 19w03b.
	Ping17ProtocolVersion19w03b int32 = 454

	// Ping17ProtocolVersion19w03a holds a protocol version (=453) for Minecraft 19w03a.
	Ping17ProtocolVersion19w03a int32 = 453

	// Ping17ProtocolVersion19w02a holds a protocol version (=452) for Minecraft 19w02a.
	Ping17ProtocolVersion19w02a int32 = 452

	// Ping17ProtocolVersion18w50a holds a protocol version (=451) for Minecraft 18w50a.
	Ping17ProtocolVersion18w50a int32 = 451

	// Ping17ProtocolVersion18w49a holds a protocol version (=450) for Minecraft 18w49a.
	Ping17ProtocolVersion18w49a int32 = 450

	// Ping17ProtocolVersion18w48b holds a protocol version (=449) for Minecraft 18w48b.
	Ping17ProtocolVersion18w48b int32 = 449

	// Ping17ProtocolVersion18w48a holds a protocol version (=448) for Minecraft 18w48a.
	Ping17ProtocolVersion18w48a int32 = 448

	// Ping17ProtocolVersion18w47b holds a protocol version (=447) for Minecraft 18w47b.
	Ping17ProtocolVersion18w47b int32 = 447

	// Ping17ProtocolVersion18w47a holds a protocol version (=446) for Minecraft 18w47a.
	Ping17ProtocolVersion18w47a int32 = 446

	// Ping17ProtocolVersion18w46a holds a protocol version (=445) for Minecraft 18w46a.
	Ping17ProtocolVersion18w46a int32 = 445

	// Ping17ProtocolVersion18w45a holds a protocol version (=444) for Minecraft 18w45a.
	Ping17ProtocolVersion18w45a int32 = 444

	// Ping17ProtocolVersion18w44a holds a protocol version (=443) for Minecraft 18w44a.
	Ping17ProtocolVersion18w44a int32 = 443

	// Ping17ProtocolVersion18w43c holds a protocol version (=442) for Minecraft 18w43c.
	Ping17ProtocolVersion18w43c int32 = 442

	// Ping17ProtocolVersion18w43b holds a protocol version (=441) for Minecraft 18w43b.
	Ping17ProtocolVersion18w43b int32 = 441

	// Ping17ProtocolVersion18w43a holds a protocol version (=441) for Minecraft 18w43a.
	Ping17ProtocolVersion18w43a int32 = 441

	// Ping17ProtocolVersion1132 holds a protocol version (=404) for Minecraft 1.13.2.
	Ping17ProtocolVersion1132 int32 = 404

	// Ping17ProtocolVersion1132pre2 holds a protocol version (=403) for Minecraft 1.13.2-pre2.
	Ping17ProtocolVersion1132pre2 int32 = 403

	// Ping17ProtocolVersion1132pre1 holds a protocol version (=402) for Minecraft 1.13.2-pre1.
	Ping17ProtocolVersion1132pre1 int32 = 402

	// Ping17ProtocolVersion1131 holds a protocol version (=401) for Minecraft 1.13.1.
	Ping17ProtocolVersion1131 int32 = 401

	// Ping17ProtocolVersion1131pre2 holds a protocol version (=400) for Minecraft 1.13.1-pre2.
	Ping17ProtocolVersion1131pre2 int32 = 400

	// Ping17ProtocolVersion1131pre1 holds a protocol version (=399) for Minecraft 1.13.1-pre1.
	Ping17ProtocolVersion1131pre1 int32 = 399

	// Ping17ProtocolVersion18w33a holds a protocol version (=398) for Minecraft 18w33a.
	Ping17ProtocolVersion18w33a int32 = 398

	// Ping17ProtocolVersion18w32a holds a protocol version (=397) for Minecraft 18w32a.
	Ping17ProtocolVersion18w32a int32 = 397

	// Ping17ProtocolVersion18w31a holds a protocol version (=396) for Minecraft 18w31a.
	Ping17ProtocolVersion18w31a int32 = 396

	// Ping17ProtocolVersion18w30b holds a protocol version (=395) for Minecraft 18w30b.
	Ping17ProtocolVersion18w30b int32 = 395

	// Ping17ProtocolVersion18w30a holds a protocol version (=394) for Minecraft 18w30a.
	Ping17ProtocolVersion18w30a int32 = 394

	// Ping17ProtocolVersion113 holds a protocol version (=393) for Minecraft 1.13.
	Ping17ProtocolVersion113 int32 = 393

	// Ping17ProtocolVersion113pre10 holds a protocol version (=392) for Minecraft 1.13-pre10.
	Ping17ProtocolVersion113pre10 int32 = 392

	// Ping17ProtocolVersion113pre9 holds a protocol version (=391) for Minecraft 1.13-pre9.
	Ping17ProtocolVersion113pre9 int32 = 391

	// Ping17ProtocolVersion113pre8 holds a protocol version (=390) for Minecraft 1.13-pre8.
	Ping17ProtocolVersion113pre8 int32 = 390

	// Ping17ProtocolVersion113pre7 holds a protocol version (=389) for Minecraft 1.13-pre7.
	Ping17ProtocolVersion113pre7 int32 = 389

	// Ping17ProtocolVersion113pre6 holds a protocol version (=388) for Minecraft 1.13-pre6.
	Ping17ProtocolVersion113pre6 int32 = 388

	// Ping17ProtocolVersion113pre5 holds a protocol version (=387) for Minecraft 1.13-pre5.
	Ping17ProtocolVersion113pre5 int32 = 387

	// Ping17ProtocolVersion113pre4 holds a protocol version (=386) for Minecraft 1.13-pre4.
	Ping17ProtocolVersion113pre4 int32 = 386

	// Ping17ProtocolVersion113pre3 holds a protocol version (=385) for Minecraft 1.13-pre3.
	Ping17ProtocolVersion113pre3 int32 = 385

	// Ping17ProtocolVersion113pre2 holds a protocol version (=384) for Minecraft 1.13-pre2.
	Ping17ProtocolVersion113pre2 int32 = 384

	// Ping17ProtocolVersion113pre1 holds a protocol version (=383) for Minecraft 1.13-pre1.
	Ping17ProtocolVersion113pre1 int32 = 383

	// Ping17ProtocolVersion18w22c holds a protocol version (=382) for Minecraft 18w22c.
	Ping17ProtocolVersion18w22c int32 = 382

	// Ping17ProtocolVersion18w22b holds a protocol version (=381) for Minecraft 18w22b.
	Ping17ProtocolVersion18w22b int32 = 381

	// Ping17ProtocolVersion18w22a holds a protocol version (=380) for Minecraft 18w22a.
	Ping17ProtocolVersion18w22a int32 = 380

	// Ping17ProtocolVersion18w21b holds a protocol version (=379) for Minecraft 18w21b.
	Ping17ProtocolVersion18w21b int32 = 379

	// Ping17ProtocolVersion18w21a holds a protocol version (=378) for Minecraft 18w21a.
	Ping17ProtocolVersion18w21a int32 = 378

	// Ping17ProtocolVersion18w20c holds a protocol version (=377) for Minecraft 18w20c.
	Ping17ProtocolVersion18w20c int32 = 377

	// Ping17ProtocolVersion18w20b holds a protocol version (=376) for Minecraft 18w20b.
	Ping17ProtocolVersion18w20b int32 = 376

	// Ping17ProtocolVersion18w20a holds a protocol version (=375) for Minecraft 18w20a.
	Ping17ProtocolVersion18w20a int32 = 375

	// Ping17ProtocolVersion18w19b holds a protocol version (=374) for Minecraft 18w19b.
	Ping17ProtocolVersion18w19b int32 = 374

	// Ping17ProtocolVersion18w19a holds a protocol version (=373) for Minecraft 18w19a.
	Ping17ProtocolVersion18w19a int32 = 373

	// Ping17ProtocolVersion18w16a holds a protocol version (=372) for Minecraft 18w16a.
	Ping17ProtocolVersion18w16a int32 = 372

	// Ping17ProtocolVersion18w15a holds a protocol version (=371) for Minecraft 18w15a.
	Ping17ProtocolVersion18w15a int32 = 371

	// Ping17ProtocolVersion18w14b holds a protocol version (=370) for Minecraft 18w14b.
	Ping17ProtocolVersion18w14b int32 = 370

	// Ping17ProtocolVersion18w14a holds a protocol version (=369) for Minecraft 18w14a.
	Ping17ProtocolVersion18w14a int32 = 369

	// Ping17ProtocolVersion18w11a holds a protocol version (=368) for Minecraft 18w11a.
	Ping17ProtocolVersion18w11a int32 = 368

	// Ping17ProtocolVersion18w10d holds a protocol version (=367) for Minecraft 18w10d.
	Ping17ProtocolVersion18w10d int32 = 367

	// Ping17ProtocolVersion18w10c holds a protocol version (=366) for Minecraft 18w10c.
	Ping17ProtocolVersion18w10c int32 = 366

	// Ping17ProtocolVersion18w10b holds a protocol version (=365) for Minecraft 18w10b.
	Ping17ProtocolVersion18w10b int32 = 365

	// Ping17ProtocolVersion18w10a holds a protocol version (=364) for Minecraft 18w10a.
	Ping17ProtocolVersion18w10a int32 = 364

	// Ping17ProtocolVersion18w09a holds a protocol version (=363) for Minecraft 18w09a.
	Ping17ProtocolVersion18w09a int32 = 363

	// Ping17ProtocolVersion18w08b holds a protocol version (=362) for Minecraft 18w08b.
	Ping17ProtocolVersion18w08b int32 = 362

	// Ping17ProtocolVersion18w08a holds a protocol version (=361) for Minecraft 18w08a.
	Ping17ProtocolVersion18w08a int32 = 361

	// Ping17ProtocolVersion18w07c holds a protocol version (=360) for Minecraft 18w07c.
	Ping17ProtocolVersion18w07c int32 = 360

	// Ping17ProtocolVersion18w07b holds a protocol version (=359) for Minecraft 18w07b.
	Ping17ProtocolVersion18w07b int32 = 359

	// Ping17ProtocolVersion18w07a holds a protocol version (=358) for Minecraft 18w07a.
	Ping17ProtocolVersion18w07a int32 = 358

	// Ping17ProtocolVersion18w06a holds a protocol version (=357) for Minecraft 18w06a.
	Ping17ProtocolVersion18w06a int32 = 357

	// Ping17ProtocolVersion18w05a holds a protocol version (=356) for Minecraft 18w05a.
	Ping17ProtocolVersion18w05a int32 = 356

	// Ping17ProtocolVersion18w03b holds a protocol version (=355) for Minecraft 18w03b.
	Ping17ProtocolVersion18w03b int32 = 355

	// Ping17ProtocolVersion18w03a holds a protocol version (=354) for Minecraft 18w03a.
	Ping17ProtocolVersion18w03a int32 = 354

	// Ping17ProtocolVersion18w02a holds a protocol version (=353) for Minecraft 18w02a.
	Ping17ProtocolVersion18w02a int32 = 353

	// Ping17ProtocolVersion18w01a holds a protocol version (=352) for Minecraft 18w01a.
	Ping17ProtocolVersion18w01a int32 = 352

	// Ping17ProtocolVersion17w50a holds a protocol version (=351) for Minecraft 17w50a.
	Ping17ProtocolVersion17w50a int32 = 351

	// Ping17ProtocolVersion17w49b holds a protocol version (=350) for Minecraft 17w49b.
	Ping17ProtocolVersion17w49b int32 = 350

	// Ping17ProtocolVersion17w49a holds a protocol version (=349) for Minecraft 17w49a.
	Ping17ProtocolVersion17w49a int32 = 349

	// Ping17ProtocolVersion17w48a holds a protocol version (=348) for Minecraft 17w48a.
	Ping17ProtocolVersion17w48a int32 = 348

	// Ping17ProtocolVersion17w47b holds a protocol version (=347) for Minecraft 17w47b.
	Ping17ProtocolVersion17w47b int32 = 347

	// Ping17ProtocolVersion17w47a holds a protocol version (=346) for Minecraft 17w47a.
	Ping17ProtocolVersion17w47a int32 = 346

	// Ping17ProtocolVersion17w46a holds a protocol version (=345) for Minecraft 17w46a.
	Ping17ProtocolVersion17w46a int32 = 345

	// Ping17ProtocolVersion17w45b holds a protocol version (=344) for Minecraft 17w45b.
	Ping17ProtocolVersion17w45b int32 = 344

	// Ping17ProtocolVersion17w45a holds a protocol version (=343) for Minecraft 17w45a.
	Ping17ProtocolVersion17w45a int32 = 343

	// Ping17ProtocolVersion17w43b holds a protocol version (=342) for Minecraft 17w43b.
	Ping17ProtocolVersion17w43b int32 = 342

	// Ping17ProtocolVersion17w43a holds a protocol version (=341) for Minecraft 17w43a.
	Ping17ProtocolVersion17w43a int32 = 341

	// Ping17ProtocolVersion1122 holds a protocol version (=340) for Minecraft 1.12.2.
	Ping17ProtocolVersion1122 int32 = 340

	// Ping17ProtocolVersion1122pre2 holds a protocol version (=339) for Minecraft 1.12.2-pre2.
	Ping17ProtocolVersion1122pre2 int32 = 339

	// Ping17ProtocolVersion1122pre1 holds a protocol version (=339) for Minecraft 1.12.2-pre1.
	Ping17ProtocolVersion1122pre1 int32 = 339

	// Ping17ProtocolVersion1121 holds a protocol version (=338) for Minecraft 1.12.1.
	Ping17ProtocolVersion1121 int32 = 338

	// Ping17ProtocolVersion1121pre1 holds a protocol version (=337) for Minecraft 1.12.1-pre1.
	Ping17ProtocolVersion1121pre1 int32 = 337

	// Ping17ProtocolVersion17w31a holds a protocol version (=336) for Minecraft 17w31a.
	Ping17ProtocolVersion17w31a int32 = 336

	// Ping17ProtocolVersion112 holds a protocol version (=335) for Minecraft 1.12.
	Ping17ProtocolVersion112 int32 = 335

	// Ping17ProtocolVersion112pre7 holds a protocol version (=334) for Minecraft 1.12-pre7.
	Ping17ProtocolVersion112pre7 int32 = 334

	// Ping17ProtocolVersion112pre6 holds a protocol version (=333) for Minecraft 1.12-pre6.
	Ping17ProtocolVersion112pre6 int32 = 333

	// Ping17ProtocolVersion112pre5 holds a protocol version (=332) for Minecraft 1.12-pre5.
	Ping17ProtocolVersion112pre5 int32 = 332

	// Ping17ProtocolVersion112pre4 holds a protocol version (=331) for Minecraft 1.12-pre4.
	Ping17ProtocolVersion112pre4 int32 = 331

	// Ping17ProtocolVersion112pre3 holds a protocol version (=330) for Minecraft 1.12-pre3.
	Ping17ProtocolVersion112pre3 int32 = 330

	// Ping17ProtocolVersion112pre2 holds a protocol version (=329) for Minecraft 1.12-pre2.
	Ping17ProtocolVersion112pre2 int32 = 329

	// Ping17ProtocolVersion112pre1 holds a protocol version (=328) for Minecraft 1.12-pre1.
	Ping17ProtocolVersion112pre1 int32 = 328

	// Ping17ProtocolVersion17w18b holds a protocol version (=327) for Minecraft 17w18b.
	Ping17ProtocolVersion17w18b int32 = 327

	// Ping17ProtocolVersion17w18a holds a protocol version (=326) for Minecraft 17w18a.
	Ping17ProtocolVersion17w18a int32 = 326

	// Ping17ProtocolVersion17w17b holds a protocol version (=325) for Minecraft 17w17b.
	Ping17ProtocolVersion17w17b int32 = 325

	// Ping17ProtocolVersion17w17a holds a protocol version (=324) for Minecraft 17w17a.
	Ping17ProtocolVersion17w17a int32 = 324

	// Ping17ProtocolVersion17w16b holds a protocol version (=323) for Minecraft 17w16b.
	Ping17ProtocolVersion17w16b int32 = 323

	// Ping17ProtocolVersion17w16a holds a protocol version (=322) for Minecraft 17w16a.
	Ping17ProtocolVersion17w16a int32 = 322

	// Ping17ProtocolVersion17w15a holds a protocol version (=321) for Minecraft 17w15a.
	Ping17ProtocolVersion17w15a int32 = 321

	// Ping17ProtocolVersion17w14a holds a protocol version (=320) for Minecraft 17w14a.
	Ping17ProtocolVersion17w14a int32 = 320

	// Ping17ProtocolVersion17w13b holds a protocol version (=319) for Minecraft 17w13b.
	Ping17ProtocolVersion17w13b int32 = 319

	// Ping17ProtocolVersion17w13a holds a protocol version (=318) for Minecraft 17w13a.
	Ping17ProtocolVersion17w13a int32 = 318

	// Ping17ProtocolVersion17w06a holds a protocol version (=317) for Minecraft 17w06a.
	Ping17ProtocolVersion17w06a int32 = 317

	// Ping17ProtocolVersion1112 holds a protocol version (=316) for Minecraft 1.11.2.
	Ping17ProtocolVersion1112 int32 = 316

	// Ping17ProtocolVersion1111 holds a protocol version (=316) for Minecraft 1.11.1.
	Ping17ProtocolVersion1111 int32 = 316

	// Ping17ProtocolVersion16w50a holds a protocol version (=316) for Minecraft 16w50a.
	Ping17ProtocolVersion16w50a int32 = 316

	// Ping17ProtocolVersion111 holds a protocol version (=315) for Minecraft 1.11.
	Ping17ProtocolVersion111 int32 = 315

	// Ping17ProtocolVersion111pre1 holds a protocol version (=314) for Minecraft 1.11-pre1.
	Ping17ProtocolVersion111pre1 int32 = 314

	// Ping17ProtocolVersion16w44a holds a protocol version (=313) for Minecraft 16w44a.
	Ping17ProtocolVersion16w44a int32 = 313

	// Ping17ProtocolVersion16w43a holds a protocol version (=313) for Minecraft 16w43a.
	Ping17ProtocolVersion16w43a int32 = 313

	// Ping17ProtocolVersion16w42a holds a protocol version (=312) for Minecraft 16w42a.
	Ping17ProtocolVersion16w42a int32 = 312

	// Ping17ProtocolVersion16w41a holds a protocol version (=311) for Minecraft 16w41a.
	Ping17ProtocolVersion16w41a int32 = 311

	// Ping17ProtocolVersion16w40a holds a protocol version (=310) for Minecraft 16w40a.
	Ping17ProtocolVersion16w40a int32 = 310

	// Ping17ProtocolVersion16w39c holds a protocol version (=309) for Minecraft 16w39c.
	Ping17ProtocolVersion16w39c int32 = 309

	// Ping17ProtocolVersion16w39b holds a protocol version (=308) for Minecraft 16w39b.
	Ping17ProtocolVersion16w39b int32 = 308

	// Ping17ProtocolVersion16w39a holds a protocol version (=307) for Minecraft 16w39a.
	Ping17ProtocolVersion16w39a int32 = 307

	// Ping17ProtocolVersion16w38a holds a protocol version (=306) for Minecraft 16w38a.
	Ping17ProtocolVersion16w38a int32 = 306

	// Ping17ProtocolVersion16w36a holds a protocol version (=305) for Minecraft 16w36a.
	Ping17ProtocolVersion16w36a int32 = 305

	// Ping17ProtocolVersion16w35a holds a protocol version (=304) for Minecraft 16w35a.
	Ping17ProtocolVersion16w35a int32 = 304

	// Ping17ProtocolVersion16w33a holds a protocol version (=303) for Minecraft 16w33a.
	Ping17ProtocolVersion16w33a int32 = 303

	// Ping17ProtocolVersion16w32b holds a protocol version (=302) for Minecraft 16w32b.
	Ping17ProtocolVersion16w32b int32 = 302

	// Ping17ProtocolVersion16w32a holds a protocol version (=301) for Minecraft 16w32a.
	Ping17ProtocolVersion16w32a int32 = 301

	// Ping17ProtocolVersion1102 holds a protocol version (=210) for Minecraft 1.10.2.
	Ping17ProtocolVersion1102 int32 = 210

	// Ping17ProtocolVersion1101 holds a protocol version (=210) for Minecraft 1.10.1.
	Ping17ProtocolVersion1101 int32 = 210

	// Ping17ProtocolVersion110 holds a protocol version (=210) for Minecraft 1.10.
	Ping17ProtocolVersion110 int32 = 210

	// Ping17ProtocolVersion110pre2 holds a protocol version (=205) for Minecraft 1.10-pre2.
	Ping17ProtocolVersion110pre2 int32 = 205

	// Ping17ProtocolVersion110pre1 holds a protocol version (=204) for Minecraft 1.10-pre1.
	Ping17ProtocolVersion110pre1 int32 = 204

	// Ping17ProtocolVersion16w21b holds a protocol version (=203) for Minecraft 16w21b.
	Ping17ProtocolVersion16w21b int32 = 203

	// Ping17ProtocolVersion16w21a holds a protocol version (=202) for Minecraft 16w21a.
	Ping17ProtocolVersion16w21a int32 = 202

	// Ping17ProtocolVersion16w20a holds a protocol version (=201) for Minecraft 16w20a.
	Ping17ProtocolVersion16w20a int32 = 201

	// Ping17ProtocolVersion194 holds a protocol version (=110) for Minecraft 1.9.4.
	Ping17ProtocolVersion194 int32 = 110

	// Ping17ProtocolVersion193 holds a protocol version (=110) for Minecraft 1.9.3.
	Ping17ProtocolVersion193 int32 = 110

	// Ping17ProtocolVersion193pre3 holds a protocol version (=110) for Minecraft 1.9.3-pre3.
	Ping17ProtocolVersion193pre3 int32 = 110

	// Ping17ProtocolVersion193pre2 holds a protocol version (=110) for Minecraft 1.9.3-pre2.
	Ping17ProtocolVersion193pre2 int32 = 110

	// Ping17ProtocolVersion193pre1 holds a protocol version (=109) for Minecraft 1.9.3-pre1.
	Ping17ProtocolVersion193pre1 int32 = 109

	// Ping17ProtocolVersion16w15b holds a protocol version (=109) for Minecraft 16w15b.
	Ping17ProtocolVersion16w15b int32 = 109

	// Ping17ProtocolVersion16w15a holds a protocol version (=109) for Minecraft 16w15a.
	Ping17ProtocolVersion16w15a int32 = 109

	// Ping17ProtocolVersion16w14a holds a protocol version (=109) for Minecraft 16w14a.
	Ping17ProtocolVersion16w14a int32 = 109

	// Ping17ProtocolVersion192 holds a protocol version (=109) for Minecraft 1.9.2.
	Ping17ProtocolVersion192 int32 = 109

	// Ping17ProtocolVersion1RVPre1 holds a protocol version (=108) for Minecraft 1.RV-Pre1.
	Ping17ProtocolVersion1RVPre1 int32 = 108

	// Ping17ProtocolVersion191 holds a protocol version (=108) for Minecraft 1.9.1.
	Ping17ProtocolVersion191 int32 = 108

	// Ping17ProtocolVersion191pre3 holds a protocol version (=108) for Minecraft 1.9.1-pre3.
	Ping17ProtocolVersion191pre3 int32 = 108

	// Ping17ProtocolVersion191pre2 holds a protocol version (=108) for Minecraft 1.9.1-pre2.
	Ping17ProtocolVersion191pre2 int32 = 108

	// Ping17ProtocolVersion191pre1 holds a protocol version (=107) for Minecraft 1.9.1-pre1.
	Ping17ProtocolVersion191pre1 int32 = 107

	// Ping17ProtocolVersion19 holds a protocol version (=107) for Minecraft 1.9.
	Ping17ProtocolVersion19 int32 = 107

	// Ping17ProtocolVersion19pre4 holds a protocol version (=106) for Minecraft 1.9-pre4.
	Ping17ProtocolVersion19pre4 int32 = 106

	// Ping17ProtocolVersion19pre3 holds a protocol version (=105) for Minecraft 1.9-pre3.
	Ping17ProtocolVersion19pre3 int32 = 105

	// Ping17ProtocolVersion19pre2 holds a protocol version (=104) for Minecraft 1.9-pre2.
	Ping17ProtocolVersion19pre2 int32 = 104

	// Ping17ProtocolVersion19pre1 holds a protocol version (=103) for Minecraft 1.9-pre1.
	Ping17ProtocolVersion19pre1 int32 = 103

	// Ping17ProtocolVersion16w07b holds a protocol version (=102) for Minecraft 16w07b.
	Ping17ProtocolVersion16w07b int32 = 102

	// Ping17ProtocolVersion16w07a holds a protocol version (=101) for Minecraft 16w07a.
	Ping17ProtocolVersion16w07a int32 = 101

	// Ping17ProtocolVersion16w06a holds a protocol version (=100) for Minecraft 16w06a.
	Ping17ProtocolVersion16w06a int32 = 100

	// Ping17ProtocolVersion16w05b holds a protocol version (=99) for Minecraft 16w05b.
	Ping17ProtocolVersion16w05b int32 = 99

	// Ping17ProtocolVersion16w05a holds a protocol version (=98) for Minecraft 16w05a.
	Ping17ProtocolVersion16w05a int32 = 98

	// Ping17ProtocolVersion16w04a holds a protocol version (=97) for Minecraft 16w04a.
	Ping17ProtocolVersion16w04a int32 = 97

	// Ping17ProtocolVersion16w03a holds a protocol version (=96) for Minecraft 16w03a.
	Ping17ProtocolVersion16w03a int32 = 96

	// Ping17ProtocolVersion16w02a holds a protocol version (=95) for Minecraft 16w02a.
	Ping17ProtocolVersion16w02a int32 = 95

	// Ping17ProtocolVersion15w51b holds a protocol version (=94) for Minecraft 15w51b.
	Ping17ProtocolVersion15w51b int32 = 94

	// Ping17ProtocolVersion15w51a holds a protocol version (=93) for Minecraft 15w51a.
	Ping17ProtocolVersion15w51a int32 = 93

	// Ping17ProtocolVersion15w50a holds a protocol version (=92) for Minecraft 15w50a.
	Ping17ProtocolVersion15w50a int32 = 92

	// Ping17ProtocolVersion15w49b holds a protocol version (=91) for Minecraft 15w49b.
	Ping17ProtocolVersion15w49b int32 = 91

	// Ping17ProtocolVersion15w49a holds a protocol version (=90) for Minecraft 15w49a.
	Ping17ProtocolVersion15w49a int32 = 90

	// Ping17ProtocolVersion15w47c holds a protocol version (=89) for Minecraft 15w47c.
	Ping17ProtocolVersion15w47c int32 = 89

	// Ping17ProtocolVersion15w47b holds a protocol version (=88) for Minecraft 15w47b.
	Ping17ProtocolVersion15w47b int32 = 88

	// Ping17ProtocolVersion15w47a holds a protocol version (=87) for Minecraft 15w47a.
	Ping17ProtocolVersion15w47a int32 = 87

	// Ping17ProtocolVersion15w46a holds a protocol version (=86) for Minecraft 15w46a.
	Ping17ProtocolVersion15w46a int32 = 86

	// Ping17ProtocolVersion15w45a holds a protocol version (=85) for Minecraft 15w45a.
	Ping17ProtocolVersion15w45a int32 = 85

	// Ping17ProtocolVersion15w44b holds a protocol version (=84) for Minecraft 15w44b.
	Ping17ProtocolVersion15w44b int32 = 84

	// Ping17ProtocolVersion15w44a holds a protocol version (=83) for Minecraft 15w44a.
	Ping17ProtocolVersion15w44a int32 = 83

	// Ping17ProtocolVersion15w43c holds a protocol version (=82) for Minecraft 15w43c.
	Ping17ProtocolVersion15w43c int32 = 82

	// Ping17ProtocolVersion15w43b holds a protocol version (=81) for Minecraft 15w43b.
	Ping17ProtocolVersion15w43b int32 = 81

	// Ping17ProtocolVersion15w43a holds a protocol version (=80) for Minecraft 15w43a.
	Ping17ProtocolVersion15w43a int32 = 80

	// Ping17ProtocolVersion15w42a holds a protocol version (=79) for Minecraft 15w42a.
	Ping17ProtocolVersion15w42a int32 = 79

	// Ping17ProtocolVersion15w41b holds a protocol version (=78) for Minecraft 15w41b.
	Ping17ProtocolVersion15w41b int32 = 78

	// Ping17ProtocolVersion15w41a holds a protocol version (=77) for Minecraft 15w41a.
	Ping17ProtocolVersion15w41a int32 = 77

	// Ping17ProtocolVersion15w40b holds a protocol version (=76) for Minecraft 15w40b.
	Ping17ProtocolVersion15w40b int32 = 76

	// Ping17ProtocolVersion15w40a holds a protocol version (=75) for Minecraft 15w40a.
	Ping17ProtocolVersion15w40a int32 = 75

	// Ping17ProtocolVersion15w39c holds a protocol version (=74) for Minecraft 15w39c.
	Ping17ProtocolVersion15w39c int32 = 74

	// Ping17ProtocolVersion15w39b holds a protocol version (=74) for Minecraft 15w39b.
	Ping17ProtocolVersion15w39b int32 = 74

	// Ping17ProtocolVersion15w39a holds a protocol version (=74) for Minecraft 15w39a.
	Ping17ProtocolVersion15w39a int32 = 74

	// Ping17ProtocolVersion15w38b holds a protocol version (=73) for Minecraft 15w38b.
	Ping17ProtocolVersion15w38b int32 = 73

	// Ping17ProtocolVersion15w38a holds a protocol version (=72) for Minecraft 15w38a.
	Ping17ProtocolVersion15w38a int32 = 72

	// Ping17ProtocolVersion15w37a holds a protocol version (=71) for Minecraft 15w37a.
	Ping17ProtocolVersion15w37a int32 = 71

	// Ping17ProtocolVersion15w36d holds a protocol version (=70) for Minecraft 15w36d.
	Ping17ProtocolVersion15w36d int32 = 70

	// Ping17ProtocolVersion15w36c holds a protocol version (=69) for Minecraft 15w36c.
	Ping17ProtocolVersion15w36c int32 = 69

	// Ping17ProtocolVersion15w36b holds a protocol version (=68) for Minecraft 15w36b.
	Ping17ProtocolVersion15w36b int32 = 68

	// Ping17ProtocolVersion15w36a holds a protocol version (=67) for Minecraft 15w36a.
	Ping17ProtocolVersion15w36a int32 = 67

	// Ping17ProtocolVersion15w35e holds a protocol version (=66) for Minecraft 15w35e.
	Ping17ProtocolVersion15w35e int32 = 66

	// Ping17ProtocolVersion15w35d holds a protocol version (=65) for Minecraft 15w35d.
	Ping17ProtocolVersion15w35d int32 = 65

	// Ping17ProtocolVersion15w35c holds a protocol version (=64) for Minecraft 15w35c.
	Ping17ProtocolVersion15w35c int32 = 64

	// Ping17ProtocolVersion15w35b holds a protocol version (=63) for Minecraft 15w35b.
	Ping17ProtocolVersion15w35b int32 = 63

	// Ping17ProtocolVersion15w35a holds a protocol version (=62) for Minecraft 15w35a.
	Ping17ProtocolVersion15w35a int32 = 62

	// Ping17ProtocolVersion15w34d holds a protocol version (=61) for Minecraft 15w34d.
	Ping17ProtocolVersion15w34d int32 = 61

	// Ping17ProtocolVersion15w34c holds a protocol version (=60) for Minecraft 15w34c.
	Ping17ProtocolVersion15w34c int32 = 60

	// Ping17ProtocolVersion15w34b holds a protocol version (=59) for Minecraft 15w34b.
	Ping17ProtocolVersion15w34b int32 = 59

	// Ping17ProtocolVersion15w34a holds a protocol version (=58) for Minecraft 15w34a.
	Ping17ProtocolVersion15w34a int32 = 58

	// Ping17ProtocolVersion15w33c holds a protocol version (=57) for Minecraft 15w33c.
	Ping17ProtocolVersion15w33c int32 = 57

	// Ping17ProtocolVersion15w33b holds a protocol version (=56) for Minecraft 15w33b.
	Ping17ProtocolVersion15w33b int32 = 56

	// Ping17ProtocolVersion15w33a holds a protocol version (=55) for Minecraft 15w33a.
	Ping17ProtocolVersion15w33a int32 = 55

	// Ping17ProtocolVersion15w32c holds a protocol version (=54) for Minecraft 15w32c.
	Ping17ProtocolVersion15w32c int32 = 54

	// Ping17ProtocolVersion15w32b holds a protocol version (=53) for Minecraft 15w32b.
	Ping17ProtocolVersion15w32b int32 = 53

	// Ping17ProtocolVersion15w32a holds a protocol version (=52) for Minecraft 15w32a.
	Ping17ProtocolVersion15w32a int32 = 52

	// Ping17ProtocolVersion15w31c holds a protocol version (=51) for Minecraft 15w31c.
	Ping17ProtocolVersion15w31c int32 = 51

	// Ping17ProtocolVersion15w31b holds a protocol version (=50) for Minecraft 15w31b.
	Ping17ProtocolVersion15w31b int32 = 50

	// Ping17ProtocolVersion15w31a holds a protocol version (=49) for Minecraft 15w31a.
	Ping17ProtocolVersion15w31a int32 = 49

	// Ping17ProtocolVersion15w14a holds a protocol version (=48) for Minecraft 15w14a.
	Ping17ProtocolVersion15w14a int32 = 48

	// Ping17ProtocolVersion189 holds a protocol version (=47) for Minecraft 1.8.9.
	Ping17ProtocolVersion189 int32 = 47

	// Ping17ProtocolVersion188 holds a protocol version (=47) for Minecraft 1.8.8.
	Ping17ProtocolVersion188 int32 = 47

	// Ping17ProtocolVersion187 holds a protocol version (=47) for Minecraft 1.8.7.
	Ping17ProtocolVersion187 int32 = 47

	// Ping17ProtocolVersion186 holds a protocol version (=47) for Minecraft 1.8.6.
	Ping17ProtocolVersion186 int32 = 47

	// Ping17ProtocolVersion185 holds a protocol version (=47) for Minecraft 1.8.5.
	Ping17ProtocolVersion185 int32 = 47

	// Ping17ProtocolVersion184 holds a protocol version (=47) for Minecraft 1.8.4.
	Ping17ProtocolVersion184 int32 = 47

	// Ping17ProtocolVersion183 holds a protocol version (=47) for Minecraft 1.8.3.
	Ping17ProtocolVersion183 int32 = 47

	// Ping17ProtocolVersion182 holds a protocol version (=47) for Minecraft 1.8.2.
	Ping17ProtocolVersion182 int32 = 47

	// Ping17ProtocolVersion182pre7 holds a protocol version (=47) for Minecraft 1.8.2-pre7.
	Ping17ProtocolVersion182pre7 int32 = 47

	// Ping17ProtocolVersion182pre6 holds a protocol version (=47) for Minecraft 1.8.2-pre6.
	Ping17ProtocolVersion182pre6 int32 = 47

	// Ping17ProtocolVersion182pre5 holds a protocol version (=47) for Minecraft 1.8.2-pre5.
	Ping17ProtocolVersion182pre5 int32 = 47

	// Ping17ProtocolVersion182pre4 holds a protocol version (=47) for Minecraft 1.8.2-pre4.
	Ping17ProtocolVersion182pre4 int32 = 47

	// Ping17ProtocolVersion182pre3 holds a protocol version (=47) for Minecraft 1.8.2-pre3.
	Ping17ProtocolVersion182pre3 int32 = 47

	// Ping17ProtocolVersion182pre2 holds a protocol version (=47) for Minecraft 1.8.2-pre2.
	Ping17ProtocolVersion182pre2 int32 = 47

	// Ping17ProtocolVersion182pre1 holds a protocol version (=47) for Minecraft 1.8.2-pre1.
	Ping17ProtocolVersion182pre1 int32 = 47

	// Ping17ProtocolVersion181 holds a protocol version (=47) for Minecraft 1.8.1.
	Ping17ProtocolVersion181 int32 = 47

	// Ping17ProtocolVersion181pre5 holds a protocol version (=47) for Minecraft 1.8.1-pre5.
	Ping17ProtocolVersion181pre5 int32 = 47

	// Ping17ProtocolVersion181pre4 holds a protocol version (=47) for Minecraft 1.8.1-pre4.
	Ping17ProtocolVersion181pre4 int32 = 47

	// Ping17ProtocolVersion181pre3 holds a protocol version (=47) for Minecraft 1.8.1-pre3.
	Ping17ProtocolVersion181pre3 int32 = 47

	// Ping17ProtocolVersion181pre2 holds a protocol version (=47) for Minecraft 1.8.1-pre2.
	Ping17ProtocolVersion181pre2 int32 = 47

	// Ping17ProtocolVersion181pre1 holds a protocol version (=47) for Minecraft 1.8.1-pre1.
	Ping17ProtocolVersion181pre1 int32 = 47

	// Ping17ProtocolVersion18 holds a protocol version (=47) for Minecraft 1.8.
	Ping17ProtocolVersion18 int32 = 47

	// Ping17ProtocolVersion18pre3 holds a protocol version (=46) for Minecraft 1.8-pre3.
	Ping17ProtocolVersion18pre3 int32 = 46

	// Ping17ProtocolVersion18pre2 holds a protocol version (=45) for Minecraft 1.8-pre2.
	Ping17ProtocolVersion18pre2 int32 = 45

	// Ping17ProtocolVersion18pre1 holds a protocol version (=44) for Minecraft 1.8-pre1.
	Ping17ProtocolVersion18pre1 int32 = 44

	// Ping17ProtocolVersion14w34d holds a protocol version (=43) for Minecraft 14w34d.
	Ping17ProtocolVersion14w34d int32 = 43

	// Ping17ProtocolVersion14w34c holds a protocol version (=42) for Minecraft 14w34c.
	Ping17ProtocolVersion14w34c int32 = 42

	// Ping17ProtocolVersion14w34b holds a protocol version (=41) for Minecraft 14w34b.
	Ping17ProtocolVersion14w34b int32 = 41

	// Ping17ProtocolVersion14w34a holds a protocol version (=40) for Minecraft 14w34a.
	Ping17ProtocolVersion14w34a int32 = 40

	// Ping17ProtocolVersion14w33c holds a protocol version (=39) for Minecraft 14w33c.
	Ping17ProtocolVersion14w33c int32 = 39

	// Ping17ProtocolVersion14w33b holds a protocol version (=38) for Minecraft 14w33b.
	Ping17ProtocolVersion14w33b int32 = 38

	// Ping17ProtocolVersion14w33a holds a protocol version (=37) for Minecraft 14w33a.
	Ping17ProtocolVersion14w33a int32 = 37

	// Ping17ProtocolVersion14w32d holds a protocol version (=36) for Minecraft 14w32d.
	Ping17ProtocolVersion14w32d int32 = 36

	// Ping17ProtocolVersion14w32c holds a protocol version (=35) for Minecraft 14w32c.
	Ping17ProtocolVersion14w32c int32 = 35

	// Ping17ProtocolVersion14w32b holds a protocol version (=34) for Minecraft 14w32b.
	Ping17ProtocolVersion14w32b int32 = 34

	// Ping17ProtocolVersion14w32a holds a protocol version (=33) for Minecraft 14w32a.
	Ping17ProtocolVersion14w32a int32 = 33

	// Ping17ProtocolVersion14w31a holds a protocol version (=32) for Minecraft 14w31a.
	Ping17ProtocolVersion14w31a int32 = 32

	// Ping17ProtocolVersion14w30c holds a protocol version (=31) for Minecraft 14w30c.
	Ping17ProtocolVersion14w30c int32 = 31

	// Ping17ProtocolVersion14w30b holds a protocol version (=30) for Minecraft 14w30b.
	Ping17ProtocolVersion14w30b int32 = 30

	// Ping17ProtocolVersion14w30a holds a protocol version (=30) for Minecraft 14w30a.
	Ping17ProtocolVersion14w30a int32 = 30

	// Ping17ProtocolVersion14w29a holds a protocol version (=29) for Minecraft 14w29a.
	Ping17ProtocolVersion14w29a int32 = 29

	// Ping17ProtocolVersion14w28b holds a protocol version (=28) for Minecraft 14w28b.
	Ping17ProtocolVersion14w28b int32 = 28

	// Ping17ProtocolVersion14w28a holds a protocol version (=27) for Minecraft 14w28a.
	Ping17ProtocolVersion14w28a int32 = 27

	// Ping17ProtocolVersion14w27b holds a protocol version (=26) for Minecraft 14w27b.
	Ping17ProtocolVersion14w27b int32 = 26

	// Ping17ProtocolVersion14w27a holds a protocol version (=26) for Minecraft 14w27a.
	Ping17ProtocolVersion14w27a int32 = 26

	// Ping17ProtocolVersion14w26c holds a protocol version (=25) for Minecraft 14w26c.
	Ping17ProtocolVersion14w26c int32 = 25

	// Ping17ProtocolVersion14w26b holds a protocol version (=24) for Minecraft 14w26b.
	Ping17ProtocolVersion14w26b int32 = 24

	// Ping17ProtocolVersion14w26a holds a protocol version (=23) for Minecraft 14w26a.
	Ping17ProtocolVersion14w26a int32 = 23

	// Ping17ProtocolVersion14w25b holds a protocol version (=22) for Minecraft 14w25b.
	Ping17ProtocolVersion14w25b int32 = 22

	// Ping17ProtocolVersion14w25a holds a protocol version (=21) for Minecraft 14w25a.
	Ping17ProtocolVersion14w25a int32 = 21

	// Ping17ProtocolVersion14w21b holds a protocol version (=20) for Minecraft 14w21b.
	Ping17ProtocolVersion14w21b int32 = 20

	// Ping17ProtocolVersion14w21a holds a protocol version (=19) for Minecraft 14w21a.
	Ping17ProtocolVersion14w21a int32 = 19

	// Ping17ProtocolVersion14w20b holds a protocol version (=18) for Minecraft 14w20b.
	Ping17ProtocolVersion14w20b int32 = 18

	// Ping17ProtocolVersion14w20a holds a protocol version (=18) for Minecraft 14w20a.
	Ping17ProtocolVersion14w20a int32 = 18

	// Ping17ProtocolVersion14w19a holds a protocol version (=17) for Minecraft 14w19a.
	Ping17ProtocolVersion14w19a int32 = 17

	// Ping17ProtocolVersion14w18b holds a protocol version (=16) for Minecraft 14w18b.
	Ping17ProtocolVersion14w18b int32 = 16

	// Ping17ProtocolVersion14w18a holds a protocol version (=16) for Minecraft 14w18a.
	Ping17ProtocolVersion14w18a int32 = 16

	// Ping17ProtocolVersion14w17a holds a protocol version (=15) for Minecraft 14w17a.
	Ping17ProtocolVersion14w17a int32 = 15

	// Ping17ProtocolVersion14w11b holds a protocol version (=14) for Minecraft 14w11b.
	Ping17ProtocolVersion14w11b int32 = 14

	// Ping17ProtocolVersion14w11a holds a protocol version (=14) for Minecraft 14w11a.
	Ping17ProtocolVersion14w11a int32 = 14

	// Ping17ProtocolVersion14w10c holds a protocol version (=13) for Minecraft 14w10c.
	Ping17ProtocolVersion14w10c int32 = 13

	// Ping17ProtocolVersion14w10b holds a protocol version (=13) for Minecraft 14w10b.
	Ping17ProtocolVersion14w10b int32 = 13

	// Ping17ProtocolVersion14w10a holds a protocol version (=13) for Minecraft 14w10a.
	Ping17ProtocolVersion14w10a int32 = 13

	// Ping17ProtocolVersion14w08a holds a protocol version (=12) for Minecraft 14w08a.
	Ping17ProtocolVersion14w08a int32 = 12

	// Ping17ProtocolVersion14w07a holds a protocol version (=11) for Minecraft 14w07a.
	Ping17ProtocolVersion14w07a int32 = 11

	// Ping17ProtocolVersion14w06b holds a protocol version (=10) for Minecraft 14w06b.
	Ping17ProtocolVersion14w06b int32 = 10

	// Ping17ProtocolVersion14w06a holds a protocol version (=10) for Minecraft 14w06a.
	Ping17ProtocolVersion14w06a int32 = 10

	// Ping17ProtocolVersion14w05b holds a protocol version (=9) for Minecraft 14w05b.
	Ping17ProtocolVersion14w05b int32 = 9

	// Ping17ProtocolVersion14w05a holds a protocol version (=9) for Minecraft 14w05a.
	Ping17ProtocolVersion14w05a int32 = 9

	// Ping17ProtocolVersion14w04b holds a protocol version (=8) for Minecraft 14w04b.
	Ping17ProtocolVersion14w04b int32 = 8

	// Ping17ProtocolVersion14w04a holds a protocol version (=7) for Minecraft 14w04a.
	Ping17ProtocolVersion14w04a int32 = 7

	// Ping17ProtocolVersion14w03b holds a protocol version (=6) for Minecraft 14w03b.
	Ping17ProtocolVersion14w03b int32 = 6

	// Ping17ProtocolVersion14w03a holds a protocol version (=6) for Minecraft 14w03a.
	Ping17ProtocolVersion14w03a int32 = 6

	// Ping17ProtocolVersion14w02c holds a protocol version (=5) for Minecraft 14w02c.
	Ping17ProtocolVersion14w02c int32 = 5

	// Ping17ProtocolVersion14w02b holds a protocol version (=5) for Minecraft 14w02b.
	Ping17ProtocolVersion14w02b int32 = 5

	// Ping17ProtocolVersion14w02a holds a protocol version (=5) for Minecraft 14w02a.
	Ping17ProtocolVersion14w02a int32 = 5

	// Ping17ProtocolVersion1710 holds a protocol version (=5) for Minecraft 1.7.10.
	Ping17ProtocolVersion1710 int32 = 5

	// Ping17ProtocolVersion1710pre4 holds a protocol version (=5) for Minecraft 1.7.10-pre4.
	Ping17ProtocolVersion1710pre4 int32 = 5

	// Ping17ProtocolVersion1710pre3 holds a protocol version (=5) for Minecraft 1.7.10-pre3.
	Ping17ProtocolVersion1710pre3 int32 = 5

	// Ping17ProtocolVersion1710pre2 holds a protocol version (=5) for Minecraft 1.7.10-pre2.
	Ping17ProtocolVersion1710pre2 int32 = 5

	// Ping17ProtocolVersion1710pre1 holds a protocol version (=5) for Minecraft 1.7.10-pre1.
	Ping17ProtocolVersion1710pre1 int32 = 5

	// Ping17ProtocolVersion179 holds a protocol version (=5) for Minecraft 1.7.9.
	Ping17ProtocolVersion179 int32 = 5

	// Ping17ProtocolVersion178 holds a protocol version (=5) for Minecraft 1.7.8.
	Ping17ProtocolVersion178 int32 = 5

	// Ping17ProtocolVersion177 holds a protocol version (=5) for Minecraft 1.7.7.
	Ping17ProtocolVersion177 int32 = 5

	// Ping17ProtocolVersion176 holds a protocol version (=5) for Minecraft 1.7.6.
	Ping17ProtocolVersion176 int32 = 5

	// Ping17ProtocolVersion176pre2 holds a protocol version (=5) for Minecraft 1.7.6-pre2.
	Ping17ProtocolVersion176pre2 int32 = 5

	// Ping17ProtocolVersion176pre1 holds a protocol version (=5) for Minecraft 1.7.6-pre1.
	Ping17ProtocolVersion176pre1 int32 = 5

	// Ping17ProtocolVersion175 holds a protocol version (=4) for Minecraft 1.7.5.
	Ping17ProtocolVersion175 int32 = 4

	// Ping17ProtocolVersion174 holds a protocol version (=4) for Minecraft 1.7.4.
	Ping17ProtocolVersion174 int32 = 4

	// Ping17ProtocolVersion173pre holds a protocol version (=4) for Minecraft 1.7.3-pre.
	Ping17ProtocolVersion173pre int32 = 4

	// Ping17ProtocolVersion13w49a holds a protocol version (=4) for Minecraft 13w49a.
	Ping17ProtocolVersion13w49a int32 = 4

	// Ping17ProtocolVersion13w48b holds a protocol version (=4) for Minecraft 13w48b.
	Ping17ProtocolVersion13w48b int32 = 4

	// Ping17ProtocolVersion13w48a holds a protocol version (=4) for Minecraft 13w48a.
	Ping17ProtocolVersion13w48a int32 = 4

	// Ping17ProtocolVersion13w47e holds a protocol version (=4) for Minecraft 13w47e.
	Ping17ProtocolVersion13w47e int32 = 4

	// Ping17ProtocolVersion13w47d holds a protocol version (=4) for Minecraft 13w47d.
	Ping17ProtocolVersion13w47d int32 = 4

	// Ping17ProtocolVersion13w47c holds a protocol version (=4) for Minecraft 13w47c.
	Ping17ProtocolVersion13w47c int32 = 4

	// Ping17ProtocolVersion13w47b holds a protocol version (=4) for Minecraft 13w47b.
	Ping17ProtocolVersion13w47b int32 = 4

	// Ping17ProtocolVersion13w47a holds a protocol version (=4) for Minecraft 13w47a.
	Ping17ProtocolVersion13w47a int32 = 4

	// Ping17ProtocolVersion172 holds a protocol version (=4) for Minecraft 1.7.2.
	Ping17ProtocolVersion172 int32 = 4

	// Ping17ProtocolVersion171pre holds a protocol version (=3) for Minecraft 1.7.1-pre.
	Ping17ProtocolVersion171pre int32 = 3

	// Ping17ProtocolVersion17pre holds a protocol version (=3) for Minecraft 1.7-pre.
	Ping17ProtocolVersion17pre int32 = 3

	// Ping17ProtocolVersion13w43a holds a protocol version (=2) for Minecraft 13w43a.
	Ping17ProtocolVersion13w43a int32 = 2

	// Ping17ProtocolVersion13w42b holds a protocol version (=1) for Minecraft 13w42b.
	Ping17ProtocolVersion13w42b int32 = 1

	// Ping17ProtocolVersion13w42a holds a protocol version (=1) for Minecraft 13w42a.
	Ping17ProtocolVersion13w42a int32 = 1

	// Ping17ProtocolVersion13w41b holds a protocol version (=0) for Minecraft 13w41b.
	Ping17ProtocolVersion13w41b int32 = 0

	// Ping17ProtocolVersion13w41a holds a protocol version (=0) for Minecraft 13w41a.
	Ping17ProtocolVersion13w41a int32 = 0
)

// PlayerEntry17 holds player sample entry from Status17 object.
type PlayerEntry17 struct {
	Nickname string
	UUID     uuid.UUID
}

// Chat17 holds arbitrary Chat data decoded from JSON. Currently untyped and unmapped to struct,
// however, this might change in future versions.
type Chat17 interface{}

type status17JsonMapping struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`

	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		Sample []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"sample"`
	} `json:"players"`

	Description Chat17 `json:"description"`
	Favicon     string `json:"favicon,omitempty"`

	PreviewsChat       bool `json:"previewsChat,omitempty"`
	EnforcesSecureChat bool `json:"enforcesSecureChat,omitempty"`
}

// Status17 holds status response returned by 1.7+ Minecraft servers.
type Status17 struct {
	VersionName     string
	ProtocolVersion int

	OnlinePlayers int
	MaxPlayers    int
	SamplePlayers []PlayerEntry17

	Description Chat17
	Icon        image.Image

	PreviewsChat       bool
	EnforcesSecureChat bool
}

// DescriptionText collects text components of Description together into normal string.
func (s *Status17) DescriptionText() string {
	componentStack := make(stack, 0, 8)
	buffer := bytes.NewBuffer(make([]byte, 0, 128))

	// Push root component to stack, whatever it is (a slice, a map or a string)
	componentStack.Push(s.Description)

	for len(componentStack) > 0 {
		// Remove topmost element from stack and get it for processing
		current, _ := componentStack.Pop()

		switch current.(type) {
		case string:
			// If component is a string, just write it to a buffer
			buffer.WriteString(current.(string))

		case []interface{}:
			// If component is a slice, push its items to stack in reverse order
			// (so that they are processed in natural order because stack is LIFO)
			current := current.([]interface{})
			for i := len(current) - 1; i >= 0; i-- {
				componentStack.Push(current[i])
			}

		case map[string]interface{}:
			// If component is an object, first its text/translate properties are handled;
			// subcomponents (aka extra) are processed last and are appended in the end of the string.
			current := current.(map[string]interface{})

			// Push extra to stack (if there is any) first as it must be processed last (stack is LIFO)
			if extra, ok := current["extra"]; ok {
				componentStack.Push(extra)
			}

			// Push component text to stack (if there is any)
			if text, ok := current["text"]; ok {
				componentStack.Push(text)
			} else if translate, ok := current["translate"]; ok {
				// If component did not contain text property, look for translate property
				// and write translate string as is, without applying "with" components or actually trying to
				// translate anything.
				componentStack.Push(translate)
			}
		}
	}

	return buffer.String()
}

// Ping17 pings 1.7+ Minecraft servers.
//
//goland:noinspection GoUnusedExportedFunction
func Ping17(host string, port int) (*Status17, error) {
	return defaultPinger.Ping17(host, port)
}

// Ping17 pings 1.7+ Minecraft servers.
func (p *Pinger) Ping17(host string, port int) (*Status17, error) {
	conn, err := p.openTCPConn(host, port)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	// Send handshake packet
	protocolVersion := p.ProtocolVersion17
	if protocolVersion == 0 {
		protocolVersion = Ping17ProtocolVersionUndefined
	}
	if err = p.ping17WriteHandshakePacket(conn, protocolVersion, host, port); err != nil {
		return nil, fmt.Errorf("could not write handshake packet: %w", err)
	}

	// Send status request packet
	if err = p.ping17WriteStatusRequestPacket(conn); err != nil {
		return nil, fmt.Errorf("could not write status request packet: %w", err)
	}

	// Read status response
	payload, err := p.ping17ReadStatusResponsePacketPayload(conn)
	if err != nil {
		return nil, fmt.Errorf("could not read response packet: %w", err)
	}

	// Parse response data from status packet
	res, err := p.ping17ParseStatusResponsePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("could not parse status from response packet: %w", err)
	}

	return res, nil
}

// Communication

func (p *Pinger) ping17WritePacket(writer io.Writer, packetID uint32, payloadData []byte) error {
	// Allocate payload buffer of size = 5 (payload length field) + payload length
	pb := bytes.NewBuffer(make([]byte, 0, 5+len(payloadData)))

	// Write packet ID as unsigned VarInt to content buffer
	b := make([]byte, 5)
	pb.Write(b[:binary.PutUvarint(b, uint64(packetID))])

	// Copy packet data to content buffer
	_, _ = pb.Write(payloadData)

	// Allocate packet buffer of size = 5 (packet ID) + payload length
	packet := bytes.NewBuffer(make([]byte, 0, 5+pb.Len()))

	// Write packet data length to packet buffer unsigned VarInt
	packet.Write(b[:binary.PutUvarint(b, uint64(pb.Len()))])

	// Write content buffer to packet buffer
	_, _ = pb.WriteTo(packet)

	_, err := packet.WriteTo(writer)
	return err
}

func (p *Pinger) ping17WriteHandshakePacket(writer io.Writer, protocol int32, host string, port int) error {
	packet := bytes.NewBuffer(make([]byte, 0, 32))

	// Write protocol version as VarInt
	b := make([]byte, 5)
	packet.Write(b[:binary.PutVarint(b, int64(protocol))])

	// Write length of hostname string as unsigned VarInt
	packet.Write(b[:binary.PutUvarint(b, uint64(len(host)))])

	// Write hostname string as byte array
	packet.Write([]byte(host))

	// Write port as unsigned short
	_ = binary.Write(packet, binary.BigEndian, uint16(port))

	// Write next state as unsigned VarInt
	packet.Write(b[:binary.PutUvarint(b, uint64(ping17NextStateStatus))])

	return p.ping17WritePacket(writer, ping17HandshakePacketID, packet.Bytes())
}

func (p *Pinger) ping17WriteStatusRequestPacket(writer io.Writer) error {
	// Write empty status request packet with only packet ID and zero length
	return p.ping17WritePacket(writer, ping17StatusRequestPacketID, nil)
}

func (p *Pinger) ping17ReadStatusResponsePacketPayload(reader io.Reader) ([]byte, error) {
	// Allocate buffer of 5 bytes (VarInt maximum length) and read packet length
	lb := make([]byte, 5)
	ln, err := reader.Read(lb)
	if err != nil {
		return nil, err
	}
	lr := bytes.NewReader(lb)

	// Read packet length as unsigned VarInt
	pl, err := binary.ReadUvarint(lr)
	if err != nil {
		return nil, err
	}

	// Read entire packet to a buffer
	pb := bytes.NewBuffer(make([]byte, 0, pl))
	pb.Write(lb[maxInt(ln-lr.Len(), 0):ln])
	if _, err = io.CopyN(pb, reader, int64(pl)-int64(lr.Len())); err != nil {
		return nil, err
	}
	pr := bytes.NewReader(pb.Bytes())

	// Read packet ID as unsigned VarInt
	id, err := binary.ReadUvarint(pr)
	if err != nil {
		return nil, err
	} else if uint32(id) != ping17StatusResponsePacketID {
		return nil, fmt.Errorf("expected packet ID %#x, but instead got %#x", ping17StatusResponsePacketID, id)
	}

	// Read status payload length
	dl, err := binary.ReadUvarint(pr)
	if err != nil {
		return nil, err
	}

	// Read packet payload
	db := bytes.NewBuffer(make([]byte, 0, dl))
	if _, err = io.CopyN(db, pr, int64(dl)); err != nil {
		return nil, err
	}

	return db.Bytes(), nil
}

// Response processing

func (p *Pinger) ping17ParseStatusResponsePayload(payload []byte) (*Status17, error) {
	// Parse JSON to struct
	var statusMapping status17JsonMapping
	if err := json.Unmarshal(payload, &statusMapping); err != nil {
		return nil, err
	}

	// Map raw status object to response struct (just these parts that can be converted right here)
	status := &Status17{
		VersionName:        statusMapping.Version.Name,
		ProtocolVersion:    statusMapping.Version.Protocol,
		OnlinePlayers:      statusMapping.Players.Online,
		MaxPlayers:         statusMapping.Players.Max,
		Description:        statusMapping.Description,
		PreviewsChat:       statusMapping.PreviewsChat,
		EnforcesSecureChat: statusMapping.EnforcesSecureChat,
	}

	// Process players sample (optionally, if UseStrict, returning on tolerable errors)
	status.SamplePlayers = make([]PlayerEntry17, len(statusMapping.Players.Sample))
	for i, entry := range statusMapping.Players.Sample {
		id, err := uuid.Parse(entry.ID)
		if err != nil {
			// Incorrect UUID is only critical in UseStrict mode; else just skip over it
			if p.UseStrict {
				return nil, fmt.Errorf("%w: invalid sample player UUID: %s", ErrInvalidStatus, err)
			}
			continue
		}

		status.SamplePlayers[i] = PlayerEntry17{entry.Name, id}
	}

	// Process icon (optionally, if UseStrict, returning on tolerable errors)
	if statusMapping.Favicon != "" {
		if !strings.HasPrefix(statusMapping.Favicon, ping17StatusImagePrefix) {
			// Incorrect prefix on favicon string only concerns us if in UseStrict mode; pass otherwise
			if p.UseStrict {
				return nil, fmt.Errorf("%w: invalid favicon data URL", ErrInvalidStatus)
			}
		} else {
			// Decode Base64 string from favicon data URL
			pngData, err := base64.StdEncoding.DecodeString(statusMapping.Favicon[len(ping17StatusImagePrefix):])
			if err != nil {
				return nil, fmt.Errorf("%w: invalid favicon image: %s", ErrInvalidStatus, err)
			}

			// Decode PNG image from binary data
			status.Icon, err = png.Decode(bytes.NewReader(pngData))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid favicon image: %s", ErrInvalidStatus, err)
			}
		}
	}

	return status, nil
}
