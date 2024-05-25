/*
   pi_sugar,
   Copyright (C) 2024  Phil Hilger

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package pi_sugar

import (
	"github.com/peergum/go-rpio/v5"
	"log"
)

type PiSugar struct {
	voltage     float64
	charge      int
	power       bool
	charging    bool
	model       int
	temperature int
	*rpio.I2cDevice
}

const (
	piSugarI2CAddress = 0x57

	powerReg         = 0x02
	temperatureReg   = 0x04
	voltageReg       = 0x22
	batteryChargeReg = 0x2a
	chargingStatusReg
)

var (
	piSugar PiSugar
)

func Init() (err error) {
	if err = rpio.Open(); err != nil {
		log.Printf("Can't open rpio %v", err)
		return err
	}

	if piSugar.I2cDevice, err = rpio.I2cBegin(rpio.I2c1, piSugarI2CAddress); err != nil {
		log.Printf("Can't start I2C %v", err)
		return err
	}
	piSugar.I2cSetSlaveAddress(0x57)
	//piSugar.I2cSetBaudrate(110000)
	return nil
}

func End() {
	piSugar.I2cEnd()
}

func NewPiSugar() (*PiSugar, error) {
	return &piSugar, nil
}

func (piSugar *PiSugar) Voltage() float64 {
	return piSugar.voltage
}

func (piSugar *PiSugar) Charge() int {
	return piSugar.charge
}

func (piSugar *PiSugar) Charging() bool {
	return piSugar.charging
}

func (piSugar *PiSugar) Power() bool {
	return piSugar.power
}

func (piSugar *PiSugar) Refresh() {
	var buf []byte = make([]byte, 2)
	code := piSugar.I2cReadRegister(temperatureReg, buf, 1)
	if code == 0 {
		piSugar.temperature = int(buf[0]) - 40
	}
	code = piSugar.I2cReadRegister(voltageReg, buf, 2)
	if code == 0 {
		piSugar.voltage = float64(uint16(buf[0])<<8|uint16(buf[1])) / 1000
	}
	code = piSugar.I2cReadRegister(batteryChargeReg, buf, 1)
	if code == 0 {
		piSugar.charge = int(buf[0])
	}
	code = piSugar.I2cReadRegister(powerReg, buf, 1)
	if code == 0 {
		piSugar.power = buf[0]&0x80 != 0
	}
	Debug("T = %dºC, V = %.3fV, B = %d%%, P = %t",
		piSugar.temperature,
		piSugar.voltage,
		piSugar.charge,
		piSugar.power)
}
