package rpio

import (
	"errors"
	"log"
)

type I2cDev int

// I2C devices.
// Only I2C1 supported for now.
const (
	I2c0 I2cDev = iota
	I2c1        // aux
	I2c2        // aux
)

const (
	controlReg             = iota /*!< BSC Master Control */
	statusReg                     /*!< BSC Master Status */
	dataLengthReg                 /*!< BSC Master Data Length */
	slaveAddressReg               /*!< BSC Master Slave Address */
	dataFifoReg                   /*!< BSC Master Data FIFO */
	clockDividerReg               /*!< BSC Master Clock Divider */
	dataDelayReg                  /*!< BSC Master Data Delay */
	clockStretchTimeoutReg        /*!< BSC Master Clock Stretch Timeout */
)

// control register masks
const (
	controlI2CEnable     = 0x00008000 /*!< I2C Enable, 0 = disabled, 1 = enabled */
	controlInterruptRX   = 0x00000400 /*!< Interrupt on RX */
	controlInterruptTX   = 0x00000200 /*!< Interrupt on TX */
	controlInterruptDone = 0x00000100 /*!< Interrupt on DONE */
	controlStartTransfer = 0x00000080 /*!< Start transfer, 1 = Start a new transfer */
	controlClearFifo1    = 0x00000020 /*!< Clear FIFO Clear */
	controlClearFifo2    = 0x00000010 /*!< Clear FIFO Clear */
	controlRead          = 0x00000001 /*!<	Read transfer */
)

/* Register masks for BSC_S */
const (
	statusClockStretchTimeout = 0x00000200 /*!< Clock stretch timeout */
	statusError               = 0x00000100 /*!< ACK error */
	statusRXFull              = 0x00000080 /*!< RXF FIFO full, 0 = FIFO is not full, 1 = FIFO is full */
	statusTXFull              = 0x00000040 /*!< TXE FIFO full, 0 = FIFO is not full, 1 = FIFO is full */
	statusRXContainsData      = 0x00000020 /*!< RXD FIFO contains data */
	statusTXAcceptsData       = 0x00000010 /*!< TXD FIFO can accept data */
	statusRXNeedsRead         = 0x00000008 /*!< RXR FIFO needs reading (full) */
	statusTXNeedsWrite        = 0x00000004 /*!< TXW FIFO needs writing (full) */
	statusTransferDone        = 0x00000002 /*!< Transfer DONE */
	statusTransferActive      = 0x00000001 /*!< Transfer Active */
)

const (
	FIFOSize = 16 /*!< BSC FIFO size */
)

/*
! \brief bcm2835I2CClockDivider

	Specifies the divider used to generate the I2C clock from the system clock.
	Clock divided is based on nominal base clock rate of 250MHz
*/
const (
	i2cClockDivider2500 = 2500 /*!< 2500 = 10us = 100 kHz */
	i2cClockDivider626  = 626  /*!< 622 = 2.504us = 399.3610 kHz */
	i2cClockDivider150  = 150  /*!< 150 = 60ns = 1.666 MHz (default at reset) */
	i2cClockDivider148  = 148  /*!< 148 = 59ns = 1.689 MHz */
)

/*
! \brief bcm2835I2CReasonCodes

	Specifies the reason codes for the bcm2835_i2c_write and bcm2835_i2c_read functions.
*/
const (
	i2cReasonOK                 = 0               /*!< Success */
	i2cErrorNACK                = 1 << (iota - 1) /*!< Received a NACK */
	i2cErrorClockStretchTimeout                   /*!< Received Clock Stretch Timeout */
	i2cErrorData                                  /*!< Not all data is sent / received */
	i2cErrorTimeout                               /*!< Time out occurred during sending */
)

var (
	I2cMapError             = errors.New("I2C registers not mapped correctly - are you root?")
	i2cByteWaitMicroseconds int64
)

// I2cBegin: Sets all pins of given I2C device to I2C mode
//
//	dev\pin | SDA | SCL |
//	I2c0    |   - |   - |
//	I2c1    |   2 |   3 |
//	I2c2    |   - |   - |
//
// It also resets I2C control register.
//
// Note that you should disable I2C interface in raspi-config first!
func I2cBegin(dev I2cDev) error {
	i2cMem[csReg] = 0 // reset i2c settings to default
	if i2cMem[csReg] == 0 {
		// this should not read only zeroes after reset -> mem map failed
		return I2cMapError
	}

	for _, pin := range getI2cPins(dev) {
		pin.Mode(I2c)
	}

	cdiv := spiMem[clockDividerReg]
	coreFreq := 250 * 1000000
	if isBCM2711() {
		coreFreq = 550 * 1000000
	}
	i2cByteWaitMicroseconds = int64(float64(cdiv) / float64(coreFreq) * float64(1000000) * 9)
	log.Printf("Microseconds wait per byte: %d", i2cByteWaitMicroseconds)

	//clearI2cTxRxFifo()
	// ensure we're staying at 100000kHz (default for the pi and pi sugar)
	setI2cDiv(i2cClockDivider2500)
	return nil
}

// I2cEnd: Sets I2C pins of given device to default (Input) mode. See I2cBegin.
func I2cEnd(dev I2cDev) {
	var pins = getI2cPins(dev)
	for _, pin := range pins {
		pin.Mode(Input)
	}
}

// I2cSpeed: Set (maximal) speed [Hz] of I2C clock.
// Param speed may be as big as 125MHz in theory, but
// only values up to 31.25MHz are considered relayable.
func I2cSpeed(speed int) {
	coreFreq := 250 * 1000000
	if isBCM2711() {
		coreFreq = 550 * 1000000
	}
	cdiv := uint32(coreFreq / speed)
	setI2cDiv(cdiv)
}

// I2cChipSelect: Select chip, one of 0, 1, 2
// for selecting slave on CE0, CE1, or CE2 pin
func I2cChipSelect(chip uint8) {
	const csMask = 3 // chip select has 2 bits

	cs := uint32(chip & csMask)

	i2cMem[csReg] = i2cMem[csReg]&^csMask | cs
}

// I2cChipSelectPolarity: Sets polarity (0/1) of active chip select
// default active=0
func I2cChipSelectPolarity(chip uint8, polarity uint8) {
	if chip > 2 {
		return
	}
	cspol := uint32(1 << (21 + chip)) // bit 21, 22 or 23 depending on chip

	if polarity == 0 { // chip select is active low
		i2cMem[csReg] &^= cspol
	} else { // chip select is active hight
		i2cMem[csReg] |= cspol
	}
}

// I2cMode: Set polarity (0/1) and phase (0/1) of i2c clock
// default polarity=0; phase=0
func I2cMode(polarity uint8, phase uint8) {
	const cpol = 1 << 3
	const cpha = 1 << 2

	if polarity == 0 { // Rest state of clock = low
		i2cMem[csReg] &^= cpol
	} else { // Rest state of clock = high
		i2cMem[csReg] |= cpol
	}

	if phase == 0 { // First SCLK transition at middle of data bit
		i2cMem[csReg] &^= cpha
	} else { // First SCLK transition at beginning of data bit
		i2cMem[csReg] |= cpha
	}
}

// I2cTransmit takes one or more bytes and send them to slave.
//
// Data received from slave are ignored.
// Use spread operator to send slice of bytes.
func I2cTransmit(data ...byte) {
	I2cExchange(append(data[:0:0], data...)) // clone data because it will be rewriten by received bytes
}

// I2cReceive receives n bytes from slave.
//
// Note that n zeroed bytes are send to slave as side effect.
func I2cReceive(n int) []byte {
	data := make([]byte, n, n)
	I2cExchange(data)
	return data
}

// I2cExchange: Transmit all bytes in data to slave
// and simultaneously receives bytes from slave to data.
//
// If you want to only send or only receive, use I2cTransmit/I2cReceive
func I2cExchange(data []byte) {
	const ta = 1 << 7   // transfer active
	const txd = 1 << 18 // tx fifo can accept data
	const rxd = 1 << 17 // rx fifo contains data
	const done = 1 << 16

	clearI2cTxRxFifo()

	// set TA = 1
	i2cMem[csReg] |= ta

	for i := range data {
		// wait for TXD
		for i2cMem[csReg]&txd == 0 {
		}
		// write bytes to I2C_FIFO
		i2cMem[fifoReg] = uint32(data[i])

		// wait for RXD
		for i2cMem[csReg]&rxd == 0 {
		}
		// read bytes from I2C_FIFO
		data[i] = byte(i2cMem[fifoReg])
	}

	// wait for DONE
	for i2cMem[csReg]&done == 0 {
	}

	// Set TA = 0
	i2cMem[csReg] &^= ta
}

// set i2c clock divider value
func setI2cDiv(div uint32) {
	const divMask = 1<<16 - 1 - 1 // cdiv have 16 bits and must be odd (for some reason)
	i2cMem[clkDivReg] = div & divMask
}

// clear both FIFOs
func clearI2cTxRxFifo() {
	const clearTxRx = 1<<5 | 1<<4
	i2cMem[csReg] |= clearTxRx
}

func getI2cPins(dev I2cDev) []Pin {
	switch dev {
	case I2c0:
		return []Pin{}
	case I2c1:
		return []Pin{2, 3}
	case I2c2:
		return []Pin{}
	default:
		return []Pin{}
	}
}
